package kit

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/config"
	"github.com/andreykaipov/goobs/api/requests/inputs"
	"github.com/andreykaipov/goobs/api/requests/scenes"
	"github.com/joho/godotenv"
)

const (
	obsSceneName     = "Palette"
	obsWindowInput   = "Resolume Window"
	obsAudioInput    = "Desktop Audio"
	obsRecordSeconds = 60
)

// Recording state
var (
	obsRecordMu       sync.Mutex
	obsRecording      bool
	obsRecordStart    time.Time
	obsRecordStopChan chan struct{}
)

func ObsProcessInfo() *ProcessInfo {

	fullpath, err := GetParam("global.obspath")
	LogIfError(err)
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("No OBS found, looking for", "path", fullpath)
		return EmptyProcessInfo()
	}

	LogOfType("obs", "Calling deleteObsSentinelpath", "fullpath", fullpath)
	// Delete the .sentinel folder to prevent OBS from showing the safe mode dialog.
	// This is needed for OBS 32.0.0+ where --disable-shutdown-check was removed.
	deleteObsSentinel()

	LogOfType("obs", "after deleteObsSentinelpath", "fullpath", fullpath)

	exe := filepath.Base(fullpath)
	pi := NewProcessInfo(exe, fullpath, "", ObsActivate)
	pi.DirPath = filepath.Dir(fullpath)
	pi.Arg = "--disable-shutdown-check"
	return pi
}

// deleteObsSentinel removes the .sentinel folder in the OBS config directory.
// OBS uses this folder to detect unclean shutdowns. By deleting it before
// starting OBS, we prevent the "safe mode" dialog from appearing.
// This is the workaround for OBS 32.0.0+ where --disable-shutdown-check was removed.
func deleteObsSentinel() {
	// OBS config is in %APPDATA%\obs-studio on Windows
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return
	}
	sentinelPath := filepath.Join(appData, "obs-studio", ".sentinel")
	if err := os.RemoveAll(sentinelPath); err != nil {
		LogIfError(err)
	} else {
		LogOfType("obs", "Deleted OBS sentinel folder", "path", sentinelPath)
	}
}

// ObsActivate is called in a goroutine, so it can block.
func ObsActivate() {
	time.Sleep(3 * time.Second)

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
}

func obsConnect() (*goobs.Client, error) {
	myenv, err := godotenv.Read(EnvFilePath())
	if err != nil {
		return nil, fmt.Errorf("cannot read .env file: %w", err)
	}
	password, ok := myenv["OBS_PASSWORD"]
	if !ok || password == "" {
		return nil, fmt.Errorf("OBS_PASSWORD not set in .env file")
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

	// Add window capture for Resolume if not present
	if !existingInputs[obsWindowInput] {
		LogOfType("obs", "Creating Resolume window capture input")
		_, err = client.Inputs.CreateInput(
			inputs.NewCreateInputParams().
				WithSceneName(obsSceneName).
				WithInputName(obsWindowInput).
				WithInputKind("window_capture").
				WithInputSettings(map[string]any{
					"window":   "Arena:Qt5152QWindowIcon:Arena.exe",
					"priority": 2, // match by executable
					"method":   2, // WGC (Windows Graphics Capture)
				}).
				WithSceneItemEnabled(true),
		)
		if err != nil {
			LogOfType("obs", "Failed to create window capture", "err", err)
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
	recordDir := filepath.Join(PaletteDataPath(), "recordings")
	if err := os.MkdirAll(recordDir, os.ModePerm); err != nil {
		LogOfType("obs", "Failed to create recordings dir", "err", err)
	}

	_, err = client.Config.SetRecordDirectory(
		config.NewSetRecordDirectoryParams().WithRecordDirectory(recordDir),
	)
	if err != nil {
		LogOfType("obs", "Failed to set record directory", "err", err)
	}

	// Set recording format to mp4 via profile parameter
	profileParams := []struct{ category, name, value string }{
		{"SimpleOutput", "RecFormat", "mp4"},
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
		LogOfType("obs", "Recording stopped", "path", resp.OutputPath)
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

// ObsRecordClip starts a timed recording for obsRecordSeconds seconds.
// Returns immediately with status info. The recording stops automatically.
func ObsRecordClip() (string, error) {
	obsRecordMu.Lock()
	defer obsRecordMu.Unlock()

	if obsRecording {
		elapsed := time.Since(obsRecordStart).Seconds()
		remaining := float64(obsRecordSeconds) - elapsed
		return fmt.Sprintf(`{"recording":true,"remaining":%.0f}`, remaining), nil
	}

	// Ensure scene is set up
	err := ObsAutoSetup()
	if err != nil {
		return "", fmt.Errorf("ObsRecordClip setup: %w", err)
	}

	// Start recording
	err = ObsCommand("recordstart")
	if err != nil {
		return "", fmt.Errorf("ObsRecordClip start: %w", err)
	}

	obsRecording = true
	obsRecordStart = time.Now()
	obsRecordStopChan = make(chan struct{})

	// Auto-stop after duration
	go func() {
		select {
		case <-time.After(time.Duration(obsRecordSeconds) * time.Second):
		case <-obsRecordStopChan:
		}

		obsRecordMu.Lock()
		wasRecording := obsRecording
		obsRecording = false
		obsRecordMu.Unlock()

		if wasRecording {
			err := ObsCommand("recordstop")
			if err != nil {
				LogOfType("obs", "ObsRecordClip auto-stop failed", "err", err)
			} else {
				LogOfType("obs", "ObsRecordClip finished")
			}
		}
	}()

	LogOfType("obs", "ObsRecordClip started", "seconds", obsRecordSeconds)
	return fmt.Sprintf(`{"recording":true,"remaining":%d}`, obsRecordSeconds), nil
}

// ObsRecordStop stops a recording in progress.
func ObsRecordStop() (string, error) {
	obsRecordMu.Lock()
	defer obsRecordMu.Unlock()

	if !obsRecording {
		return `{"recording":false}`, nil
	}

	obsRecording = false
	close(obsRecordStopChan)

	return `{"recording":false}`, nil
}

// ObsRecordStatus returns the current recording state.
func ObsRecordStatus() (string, error) {
	obsRecordMu.Lock()
	defer obsRecordMu.Unlock()

	if !obsRecording {
		return `{"recording":false,"remaining":0}`, nil
	}

	elapsed := time.Since(obsRecordStart).Seconds()
	remaining := float64(obsRecordSeconds) - elapsed
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf(`{"recording":true,"remaining":%.0f}`, remaining), nil
}
