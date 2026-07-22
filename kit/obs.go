package kit

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/config"
	"github.com/andreykaipov/goobs/api/requests/inputs"
	"github.com/andreykaipov/goobs/api/requests/scenes"
)

const (
	obsSceneName     = "Palette"
	obsWindowInput   = "Resolume Window"
	obsAudioInput    = "Desktop Audio"
	obsMicAuxInput   = "Mic/Aux"
	obsRecordSeconds = 60
)

// Recording state
var (
	obsRecordMu       sync.Mutex
	obsRecording      bool
	obsRecordStart    time.Time
	obsRecordStopChan chan struct{}
)

type OBSRecordState struct {
	Recording bool    `json:"recording"`
	Remaining float64 `json:"remaining"`
}

func ObsProcessInfo() *ProcessInfo {

	fullpath, err := GetParam("global.obspath")
	LogIfError(err)
	if fullpath != "" && !FileExists(fullpath) {
		// The configured default is a Windows path, so fall back to the
		// usual install locations for this OS before giving up.
		found := firstExistingPath(ObsCandidatePaths())
		if found == "" {
			LogWarn("No OBS found, looking for", "path", fullpath)
			return EmptyProcessInfo()
		}
		LogInfo("OBS found at its default install location", "path", found)
		fullpath = found
	}

	exe := filepath.Base(fullpath)
	pi := NewProcessInfo(exe, fullpath, "", ObsActivate)
	pi.DirPath = filepath.Dir(fullpath)
	pi.Arg = "--disable-shutdown-check"
	pi.BeforeLaunch = deleteObsSentinel
	return pi
}

// deleteObsSentinel removes the .sentinel folder in the OBS config directory.
// OBS uses this folder to detect unclean shutdowns. By deleting it before
// starting OBS, we prevent the "safe mode" dialog from appearing.
// This is the workaround for OBS 32.0.0+ where --disable-shutdown-check was removed.
func deleteObsSentinel() error {
	// OBS config lives in %APPDATA%\obs-studio on Windows and
	// ~/Library/Application Support/obs-studio on macOS.
	configDir := ObsConfigDir()
	if configDir == "" {
		return fmt.Errorf("cannot locate OBS sentinel: no OBS config directory for this system")
	}
	sentinelPath := filepath.Join(configDir, ".sentinel")
	if err := os.RemoveAll(sentinelPath); err != nil {
		return fmt.Errorf("remove OBS sentinel %q: %w", sentinelPath, err)
	}
	// Do not launch OBS if a stale marker remains. Otherwise OBS can present the
	// safe-mode chooser even though Palette attempted the cleanup.
	if _, err := os.Lstat(sentinelPath); err == nil {
		return fmt.Errorf("OBS sentinel still exists after removal: %q", sentinelPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("verify OBS sentinel removal %q: %w", sentinelPath, err)
	}
	LogOfType("obs", "Cleared OBS sentinel before launch", "path", sentinelPath)
	return nil
}

// ObsActivate is called in a goroutine, so it can block.
func ObsActivate() {

	// Wait for OBS's websocket to accept connections rather than sleeping
	// a fixed time; a cold OBS start can take well over 3 seconds, and the
	// status push below is what reveals the RECORD button in the GUI.
	deadline := time.Now().Add(60 * time.Second)
	for !ObsIsRunning() {
		if time.Now().After(deadline) {
			LogWarn("ObsActivate: OBS websocket never became ready")
			break
		}
		time.Sleep(time.Second)
	}

	// Auto-setup the Palette scene if needed
	err := ObsAutoSetup()
	if err != nil {
		LogOfType("obs", "ObsAutoSetup failed", "err", err)
	}

	stream, err := GetParamBool("global.obsstream")
	LogIfError(err)
	if err == nil && stream {
		LogOfType("obs", "ObsActivate calling streamstart")
		err = ObsCommand("streamstart")
		LogIfError(err)
	}

	// OBS (and its WebSocket) is now up. Push a status update so any open GUI
	// reveals the RECORD button without needing a reload.
	NotifyStatusChanged()
	NotifyOBSRecordChanged()
}

func obsConnect() (*goobs.Client, error) {
	// OBS_PASSWORD is read from the env file (.palette/.env), falling back to
	// the OS environment variable. If it's not set anywhere we log a warning
	// and still attempt the connection with an empty password: this works if
	// OBS has authentication disabled; if OBS requires auth, goobs will fail
	// the handshake instead.
	password := EnvLookup("OBS_PASSWORD")
	if password == "" {
		LogWarn("OBS: OBS_PASSWORD not set in env file or environment, connecting with empty password")
	}
	return goobs.New("localhost:4455", goobs.WithPassword(password))
}

// ObsAutoSetup ensures the Palette scene exists with the correct sources and recording settings.
func ObsAutoSetup() error {
	client, err := obsConnect()
	if err != nil {
		return fmt.Errorf("ObsAutoSetup connect: %w", err)
	}
	defer client.Disconnect()

	// Check if our scene already exists
	sceneList, err := client.Scenes.GetSceneList()
	if err != nil {
		return fmt.Errorf("ObsAutoSetup GetSceneList: %w", err)
	}

	sceneExists := false
	for _, s := range sceneList.Scenes {
		if s.SceneName == obsSceneName {
			sceneExists = true
			break
		}
	}

	if !sceneExists {
		LogOfType("obs", "Creating Palette scene")
		_, err = client.Scenes.CreateScene(
			scenes.NewCreateSceneParams().WithSceneName(obsSceneName),
		)
		if err != nil {
			return fmt.Errorf("ObsAutoSetup CreateScene: %w", err)
		}
	}

	// Switch to our scene
	_, err = client.Scenes.SetCurrentProgramScene(
		scenes.NewSetCurrentProgramSceneParams().WithSceneName(obsSceneName),
	)
	if err != nil {
		return fmt.Errorf("ObsAutoSetup SetCurrentProgramScene: %w", err)
	}

	// Check existing inputs to avoid duplicates
	inputList, err := client.Inputs.GetInputList()
	if err != nil {
		return fmt.Errorf("ObsAutoSetup GetInputList: %w", err)
	}
	existingInputs := make(map[string]bool)
	for _, inp := range inputList.Inputs {
		existingInputs[inp.InputName] = true
	}

	// Palette recordings intentionally contain desktop audio only. Mic/Aux is
	// a global OBS input, so it can be recorded even though it is not a source
	// in the Palette scene. Re-apply the mute on every OBS startup in case it
	// was changed during a previous session.
	if existingInputs[obsMicAuxInput] {
		LogOfType("obs", "Muting Mic/Aux for Palette recording")
		_, err = client.Inputs.SetInputMute(
			inputs.NewSetInputMuteParams().
				WithInputName(obsMicAuxInput).
				WithInputMuted(true),
		)
		if err != nil {
			return fmt.Errorf("ObsAutoSetup mute Mic/Aux: %w", err)
		}
	}

	resolumeWindowSettings := obsResolumeWindowSettings()

	// Add or update window capture for Resolume. The executable may be
	// Avenue.exe or Arena.exe depending on the installed Resolume edition.
	if !existingInputs[obsWindowInput] {
		LogOfType("obs", "Creating Resolume window capture input")
		_, err = client.Inputs.CreateInput(
			inputs.NewCreateInputParams().
				WithSceneName(obsSceneName).
				WithInputName(obsWindowInput).
				WithInputKind("window_capture").
				WithInputSettings(resolumeWindowSettings).
				WithSceneItemEnabled(true),
		)
		if err != nil {
			LogOfType("obs", "Failed to create window capture", "err", err)
		}
	} else {
		LogOfType("obs", "Updating Resolume window capture input")
		_, err = client.Inputs.SetInputSettings(
			inputs.NewSetInputSettingsParams().
				WithInputName(obsWindowInput).
				WithInputSettings(resolumeWindowSettings).
				WithOverlay(true),
		)
		if err != nil {
			LogOfType("obs", "Failed to update window capture", "err", err)
		}
	}

	// Add desktop audio capture if not present
	if !existingInputs[obsAudioInput] {
		LogOfType("obs", "Creating desktop audio input")
		_, err = client.Inputs.CreateInput(
			inputs.NewCreateInputParams().
				WithSceneName(obsSceneName).
				WithInputName(obsAudioInput).
				WithInputKind("wasapi_output_capture").
				WithInputSettings(map[string]any{
					"device_id": "default",
				}).
				WithSceneItemEnabled(true),
		)
		if err != nil {
			LogOfType("obs", "Failed to create audio capture", "err", err)
		}
	}

	// Configure recording output settings
	recordDir := obsRecordDir()
	if err := os.MkdirAll(recordDir, os.ModePerm); err != nil {
		LogOfType("obs", "Failed to create recordings dir", "err", err)
	}

	_, err = client.Config.SetRecordDirectory(
		config.NewSetRecordDirectoryParams().WithRecordDirectory(recordDir),
	)
	if err != nil {
		LogOfType("obs", "Failed to set record directory", "err", err)
	}

	// Set recording format via profile parameter. OBS 30.2+ supports "hybrid
	// mp4", which stays playable even if OBS dies mid-recording; plain mp4
	// only writes its index at finalization and is corrupted by a hard kill.
	recFormat := "mp4"
	if version, err := client.General.GetVersion(); err == nil && obsVersionAtLeast(version.ObsVersion, 30, 2) {
		recFormat = "hybrid_mp4"
	}
	LogOfType("obs", "Setting OBS recording format", "format", recFormat)
	profileParams := []struct{ category, name, value string }{
		{"SimpleOutput", "RecFormat", recFormat},
		{"SimpleOutput", "RecQuality", "Stream"},
	}
	for _, pp := range profileParams {
		_, err = client.Config.SetProfileParameter(
			config.NewSetProfileParameterParams().
				WithParameterCategory(pp.category).
				WithParameterName(pp.name).
				WithParameterValue(pp.value),
		)
		if err != nil {
			LogOfType("obs", "Failed to set profile param", "category", pp.category, "name", pp.name, "err", err)
		}
	}

	LogOfType("obs", "OBS auto-setup complete", "scene", obsSceneName, "recordDir", recordDir)
	return nil
}

// obsVersionAtLeast reports whether an OBS version string like "30.2.3" is at
// least major.minor. Any parse failure returns false.
func obsVersionAtLeast(version string, major, minor int) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	min, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	return maj > major || (maj == major && min >= minor)
}

// obsGracefulRecordStop makes a best-effort attempt to stop an in-progress
// recording before OBS is killed, so OBS finalizes the file instead of
// leaving a corrupt mp4. Errors are ignored — obs-websocket returns an error
// when no recording is active, and shutdown must proceed regardless. The
// recordstop runs against a deadline so a hung OBS can't stall shutdown.
func obsGracefulRecordStop() {
	if !ObsIsRunning() {
		return
	}
	errChan := make(chan error, 1)
	go func() {
		errChan <- ObsCommand("recordstop")
	}()
	select {
	case err := <-errChan:
		if err != nil {
			LogOfType("obs", "obsGracefulRecordStop: no active recording to stop", "err", err)
			return
		}
	case <-time.After(3 * time.Second):
		LogOfType("obs", "obsGracefulRecordStop: timed out waiting for recordstop")
		return
	}
	LogOfType("obs", "obsGracefulRecordStop: finalized in-progress recording before OBS is killed")
	// Give OBS a moment to finish writing the file before it gets killed.
	time.Sleep(2 * time.Second)
}

func obsResolumeWindowSettings() map[string]any {
	appName := obsResolumeAppName()
	return map[string]any{
		"window":   fmt.Sprintf("%s:Qt5152QWindowIcon:%s.exe", appName, appName),
		"priority": 2, // match by executable
		"method":   2, // WGC (Windows Graphics Capture)
	}
}

func obsResolumeAppName() string {
	configuredPath, err := GetParam("global.resolumepath")
	if err == nil && configuredPath != "" && FileExists(configuredPath) {
		if appName := resolumeAppNameFromPath(configuredPath); appName != "" {
			return appName
		}
	}

	for _, path := range []string{
		"C:/Program Files/Resolume Avenue/Avenue.exe",
		"C:/Program Files/Resolume Arena/Arena.exe",
	} {
		if FileExists(path) {
			if appName := resolumeAppNameFromPath(path); appName != "" {
				return appName
			}
		}
	}

	if err == nil && configuredPath != "" {
		if appName := resolumeAppNameFromPath(configuredPath); appName != "" {
			return appName
		}
	}
	return "Avenue"
}

func resolumeAppNameFromPath(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	switch {
	case strings.EqualFold(base, "Arena"):
		return "Arena"
	case strings.EqualFold(base, "Avenue"):
		return "Avenue"
	default:
		return ""
	}
}

// ObsIsRunning probes the OBS WebSocket port with a short TCP dial.
// Returns true iff something is accepting connections on localhost:4455 —
// a quick, goobs-free check used by the web UI to decide whether to show
// the Record button.
func ObsIsRunning() bool {
	conn, err := net.DialTimeout("tcp", "localhost:4455", 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func ObsCommand(cmd string) error {

	client, err := obsConnect()
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Disconnect()
	}()

	switch cmd {
	case "status":
		version, err := client.General.GetVersion()
		if err != nil {
			return err
		}
		recordStatus, err := client.Record.GetRecordStatus()
		if err != nil {
			return err
		}
		streamStatus, err := client.Stream.GetStreamStatus()
		if err != nil {
			return err
		}
		fmt.Printf("Streaming active: %v\n", streamStatus.OutputActive)
		fmt.Printf("Recording active: %v\n", recordStatus.OutputActive)
		fmt.Printf("OBS Studio version: %s\n", version.ObsVersion)

	case "recordstart":
		_, err := client.Record.StartRecord()
		return err
	case "recordstop":
		resp, err := client.Record.StopRecord()
		if err != nil {
			return err
		}
		path := renameRecordingToGeneratedName(resp.OutputPath)
		LogOfType("obs", "Recording stopped", "path", path)
		return nil

	case "streamstart":
		_, err := client.Stream.StartStream()
		return err
	case "streamstop":
		_, err := client.Stream.StopStream()
		return err

	case "setup":
		return ObsAutoSetup()

	default:
		return fmt.Errorf("unknown obs command: %s", cmd)
	}

	return nil
}

// obsSetRecordFilename tells OBS to name its next recording with a freshly
// generated three-word name (OBS appends the container extension itself).
// Generated names are lowercase letters and underscores only, so OBS can't
// misread them as % format specifiers. Returns the name it set.
func obsSetRecordFilename() (string, error) {
	name, err := UniqueRandomName(func(name string) bool {
		return FileExists(filepath.Join(obsRecordDir(), name+".mp4"))
	})
	if err != nil {
		return "", err
	}

	client, err := obsConnect()
	if err != nil {
		return "", err
	}
	defer client.Disconnect()

	// FilenameFormatting lives under [Output] in the profile's basic.ini,
	// not under SimpleOutput like the RecFormat/RecQuality settings.
	_, err = client.Config.SetProfileParameter(
		config.NewSetProfileParameterParams().
			WithParameterCategory("Output").
			WithParameterName("FilenameFormatting").
			WithParameterValue(name),
	)
	if err != nil {
		return "", err
	}
	return name, nil
}

// ObsRecordClip starts a timed recording for obsRecordSeconds seconds.
// Returns immediately with status info. The recording stops automatically.
func ObsRecordClip() (string, error) {
	obsRecordMu.Lock()

	if obsRecording {
		elapsed := time.Since(obsRecordStart).Seconds()
		remaining := float64(obsRecordSeconds) - elapsed
		obsRecordMu.Unlock()
		return fmt.Sprintf(`{"recording":true,"remaining":%.0f}`, remaining), nil
	}

	// Ensure scene is set up
	err := ObsAutoSetup()
	if err != nil {
		obsRecordMu.Unlock()
		return "", fmt.Errorf("ObsRecordClip setup: %w", err)
	}

	// Name the recording up front so the file is born with its generated
	// name; the rename at stop remains only as a fallback.
	name, err := obsSetRecordFilename()
	if err != nil {
		LogOfType("obs", "ObsRecordClip could not preset recording filename", "err", err)
	}

	// Start recording
	err = ObsCommand("recordstart")
	if err != nil {
		obsRecordMu.Unlock()
		return "", fmt.Errorf("ObsRecordClip start: %w", err)
	}

	obsRecording = true
	obsRecordStart = time.Now()
	stopChan := make(chan struct{})
	obsRecordStopChan = stopChan

	// Auto-stop after duration
	go func(stopChan <-chan struct{}) {
		select {
		case <-time.After(time.Duration(obsRecordSeconds) * time.Second):
		case <-stopChan:
			return
		}

		obsRecordMu.Lock()
		wasRecording := obsRecording
		startTime := obsRecordStart
		if wasRecording {
			obsRecording = false
			obsRecordStopChan = nil
		}
		obsRecordMu.Unlock()

		if !wasRecording {
			return
		}

		err := ObsCommand("recordstop")
		elapsed := time.Since(startTime).Round(time.Millisecond)
		if err != nil {
			LogOfType("obs", "OBS recording stopped (timer elapsed) but recordstop failed", "elapsed", elapsed, "err", err)
		} else {
			LogOfType("obs", "OBS recording stopped: timer elapsed", "elapsed", elapsed)
		}
		NotifyOBSRecordChanged()
	}(stopChan)

	LogOfType("obs", "ObsRecordClip started", "seconds", obsRecordSeconds, "name", name)
	obsRecordMu.Unlock()
	NotifyOBSRecordChanged()
	return fmt.Sprintf(`{"recording":true,"remaining":%d}`, obsRecordSeconds), nil
}

// ObsRecordStop stops a recording in progress.
func ObsRecordStop() (string, error) {
	obsRecordMu.Lock()

	if !obsRecording {
		obsRecordMu.Unlock()
		return `{"recording":false}`, nil
	}

	obsRecording = false
	startTime := obsRecordStart
	stopChan := obsRecordStopChan
	obsRecordStopChan = nil
	if stopChan != nil {
		close(stopChan)
	}
	obsRecordMu.Unlock()

	err := ObsCommand("recordstop")
	elapsed := time.Since(startTime).Round(time.Millisecond)
	if err != nil {
		LogOfType("obs", "OBS recording stopped (early) but recordstop failed", "elapsed", elapsed, "err", err)
		NotifyOBSRecordChanged()
		return "", fmt.Errorf("ObsRecordStop stop: %w", err)
	}

	LogOfType("obs", "OBS recording stopped: stopped early", "elapsed", elapsed)
	NotifyOBSRecordChanged()
	return `{"recording":false}`, nil
}

// ObsRecordStatus returns the current recording state.
func ObsRecordStatus() (string, error) {
	status := ObsRecordStatusSnapshot()
	return fmt.Sprintf(`{"recording":%t,"remaining":%.0f}`, status.Recording, status.Remaining), nil
}

func ObsRecordStatusSnapshot() OBSRecordState {
	obsRecordMu.Lock()
	defer obsRecordMu.Unlock()

	if !obsRecording {
		return OBSRecordState{Recording: false, Remaining: 0}
	}

	elapsed := time.Since(obsRecordStart).Seconds()
	remaining := float64(obsRecordSeconds) - elapsed
	if remaining < 0 {
		remaining = 0
	}
	return OBSRecordState{Recording: true, Remaining: remaining}
}

// obsRecordDir is the directory OBS is configured to write recordings to.
func obsRecordDir() string {
	return filepath.Join(PaletteDataPath(), "recordings")
}

// renameRecordingToGeneratedName renames a just-finished OBS recording from
// its default timestamp name to a memorable, anonymous three-word name (e.g.
// "gentle_otter_sunrise.mp4"), keeping it in the same directory and preserving
// the extension. On any failure it logs and returns the original path so the
// recording is never lost.
func renameRecordingToGeneratedName(outputPath string) string {
	if outputPath == "" {
		return outputPath
	}
	dir := filepath.Dir(outputPath)
	ext := filepath.Ext(outputPath)

	// The filename is normally preset at record start; if it already has a
	// generated-style name, there is nothing to do.
	if IsGeneratedName(strings.TrimSuffix(filepath.Base(outputPath), ext)) {
		return outputPath
	}

	name, err := UniqueRandomName(func(name string) bool {
		return FileExists(filepath.Join(dir, name+ext))
	})
	if err != nil {
		LogOfType("obs", "Recording keeping default name; could not generate a unique one", "err", err, "path", outputPath)
		return outputPath
	}

	newPath := filepath.Join(dir, name+ext)
	if err := os.Rename(outputPath, newPath); err != nil {
		LogOfType("obs", "Recording keeping default name; rename failed", "err", err, "from", outputPath, "to", newPath)
		return outputPath
	}
	return newPath
}

// recordingPath validates a recording filename from the UI or CLI and
// returns its full path.  The name must be a plain filename (no directory
// components) of a file that exists in the recordings directory.
func recordingPath(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("bad recording name: %s", name)
	}
	path := filepath.Join(obsRecordDir(), name)
	if !FileExists(path) {
		return "", fmt.Errorf("no such recording: %s", name)
	}
	return path, nil
}

// ObsRecordDelete deletes one file from the recordings directory.
func ObsRecordDelete(name string) error {
	path, err := recordingPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("ObsRecordDelete: %w", err)
	}
	LogInfo("Recording deleted", "name", name)
	return nil
}

// OBSRecordingFile describes one recorded file for the recordings list UI.
type OBSRecordingFile struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	ModTime  string  `json:"modtime"`
	Duration float64 `json:"duration"` // seconds; 0 if unknown
}

// ObsRecordList returns a JSON array of the files in the recordings directory,
// most-recent first. A missing directory is not an error — it returns "[]".
func ObsRecordList() (string, error) {
	entries, err := os.ReadDir(obsRecordDir())
	if err != nil {
		if os.IsNotExist(err) {
			return "[]", nil
		}
		return "", fmt.Errorf("ObsRecordList: %w", err)
	}

	files := []OBSRecordingFile{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		rec := OBSRecordingFile{
			Name:    e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".mp4") {
			if secs, ok := mp4DurationSeconds(filepath.Join(obsRecordDir(), e.Name())); ok {
				rec.Duration = secs
			}
		}
		files = append(files, rec)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime > files[j].ModTime
	})

	b, err := json.Marshal(files)
	if err != nil {
		return "", fmt.Errorf("ObsRecordList marshal: %w", err)
	}
	return string(b), nil
}

// mp4DurationSeconds reads the movie duration from an MP4/QuickTime file's
// moov/mvhd box. Returns (seconds, true) on success, or (0, false) if the file
// can't be read or isn't a well-formed MP4. It does not depend on any external
// tool (ffprobe etc.).
func mp4DurationSeconds(path string) (float64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, false
	}
	fileEnd := info.Size()

	moovStart, moovEnd, ok := findMp4Box(f, 0, fileEnd, "moov")
	if !ok {
		return 0, false
	}
	mvhdStart, mvhdEnd, ok := findMp4Box(f, moovStart, moovEnd, "mvhd")
	if !ok {
		return 0, false
	}

	buf := make([]byte, mvhdEnd-mvhdStart)
	if _, err := f.ReadAt(buf, mvhdStart); err != nil {
		return 0, false
	}

	// mvhd payload: version(1) flags(3), then version-dependent fields.
	var timescale, duration uint64
	switch buf[0] {
	case 1:
		if len(buf) < 32 {
			return 0, false
		}
		timescale = uint64(binary.BigEndian.Uint32(buf[20:24]))
		duration = binary.BigEndian.Uint64(buf[24:32])
	default: // version 0
		if len(buf) < 20 {
			return 0, false
		}
		timescale = uint64(binary.BigEndian.Uint32(buf[12:16]))
		duration = uint64(binary.BigEndian.Uint32(buf[16:20]))
	}
	if timescale == 0 {
		return 0, false
	}
	return float64(duration) / float64(timescale), true
}

// findMp4Box scans the MP4 box list in [start, end) of r for a box of the given
// 4-character type, returning the byte range [payloadStart, payloadEnd) of its
// contents. Handles 64-bit largesize and box-extends-to-end (size 0).
func findMp4Box(r io.ReaderAt, start, end int64, boxType string) (int64, int64, bool) {
	off := start
	hdr := make([]byte, 8)
	for off+8 <= end {
		if _, err := r.ReadAt(hdr, off); err != nil {
			return 0, 0, false
		}
		size := int64(binary.BigEndian.Uint32(hdr[0:4]))
		typ := string(hdr[4:8])
		payloadOff := off + 8
		var boxEnd int64
		switch size {
		case 1:
			lb := make([]byte, 8)
			if _, err := r.ReadAt(lb, off+8); err != nil {
				return 0, 0, false
			}
			size = int64(binary.BigEndian.Uint64(lb))
			payloadOff = off + 16
			boxEnd = off + size
		case 0:
			boxEnd = end
		default:
			boxEnd = off + size
		}
		// boxEnd == payloadOff is a valid header-only (empty) box; only reject
		// a box whose declared size is smaller than its header, or overruns.
		if boxEnd < payloadOff || boxEnd > end {
			return 0, 0, false
		}
		if typ == boxType {
			return payloadOff, boxEnd, true
		}
		off = boxEnd
	}
	return 0, 0, false
}
