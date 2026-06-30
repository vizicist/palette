package kit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ss "github.com/vizicist/palette/pkg/samplesplitter"
)

const samplesplitterPort = 9876
const samplesplitterMidiPort = "16. Internal MIDI"
const (
	defaultSamplePlaybackReverbWet    = 0.0
	defaultSamplePlaybackReverbLength = ss.DefaultReverbLength
)

var samplePlaybackService struct {
	mutex   sync.Mutex
	service *ss.Service
	cancel  context.CancelFunc
}

func StartSamplePlaybackService() error {
	if !samplePlaybackProcessAllowed() {
		return fmt.Errorf("sample playback is disabled in %s mode", CurrentMode())
	}
	samplePlaybackService.mutex.Lock()
	defer samplePlaybackService.mutex.Unlock()
	if samplePlaybackService.service != nil && samplePlaybackService.service.Running() {
		if !IsBSSMode() {
			return syncProSamplePlaybackSamples(samplePlaybackService.service)
		}
		return nil
	}
	runtimeDir := SamplesplitterRuntimeDir()
	config := ss.DefaultConfig()
	config.Port = samplesplitterPort
	config.MP3Dir = SamplesplitterMP3Dir(runtimeDir)
	if !IsBSSMode() {
		dir, err := firstProSamplePlaybackDir()
		if err != nil {
			return err
		}
		config.MP3Dir = dir
	}
	config.MIDIPortName = ""
	config.FFmpegPath = ss.FindFFmpeg(runtimeDir)
	config.Compressed = samplePlaybackCompressed()
	config.ReverbWet = samplePlaybackReverbWet()
	config.ReverbLength = samplePlaybackReverbLength()
	if words, err := samplePlaybackWords(); err == nil {
		config.DefaultWords = words
	}
	service, err := ss.NewService(ss.ServiceOptions{
		Config:       config,
		StaticDir:    ss.ResolveStaticDir(runtimeDir),
		EnableMIDI:   false,
		EnableHTTP:   true,
		SelectOutput: true,
	})
	if err != nil {
		return err
	}
	if !IsBSSMode() {
		if err := syncProSamplePlaybackSamples(service); err != nil {
			_ = service.Close()
			return err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := service.Start(ctx); err != nil {
		cancel()
		_ = service.Close()
		return err
	}
	samplePlaybackService.service = service
	samplePlaybackService.cancel = cancel
	LogInfo("StartSamplePlaybackService", "mp3Dir", config.MP3Dir, "ffmpeg", config.FFmpegPath)
	logSelectedSamplePlaybackFiles("StartSamplePlaybackService", service)
	return nil
}

func samplePlaybackCompressed() bool {
	enabled, err := GetParamBool("global.sampleplaybackcompressed")
	if err == nil {
		return enabled
	}
	LogIfError(err)
	return false
}

func samplePlaybackWords() (int, error) {
	words, err := GetParamInt("global.sampleplaybackwords")
	if err == nil {
		return clampSamplePlaybackWords(words), nil
	}
	return 2, err
}

func samplePlaybackReverbWet() float64 {
	wet, err := getSamplePlaybackFloatParam("global.sampleplaybackreverb", defaultSamplePlaybackReverbWet)
	if err != nil {
		LogIfError(err)
	}
	return clampSamplePlaybackReverbWet(wet)
}

func clampSamplePlaybackReverbWet(wet float64) float64 {
	if wet < 0 {
		return 0
	}
	if wet > 1 {
		return 1
	}
	return wet
}

func samplePlaybackReverbLength() float64 {
	length, err := getSamplePlaybackFloatParam("global.sampleplaybackreverblength", defaultSamplePlaybackReverbLength)
	if err != nil {
		LogIfError(err)
	}
	return clampSamplePlaybackReverbLength(length)
}

func clampSamplePlaybackReverbLength(length float64) float64 {
	if length < ss.MinReverbLength {
		return ss.MinReverbLength
	}
	if length > ss.MaxReverbLength {
		return ss.MaxReverbLength
	}
	return length
}

func clampSamplePlaybackWords(words int) int {
	if words < 1 {
		return 1
	}
	if words > 16 {
		return 16
	}
	return words
}

func StopSamplePlaybackService() {
	samplePlaybackService.mutex.Lock()
	service := samplePlaybackService.service
	cancel := samplePlaybackService.cancel
	samplePlaybackService.service = nil
	samplePlaybackService.cancel = nil
	samplePlaybackService.mutex.Unlock()
	if cancel != nil {
		cancel()
	}
	if service != nil {
		_ = service.Close()
	}
}

func SamplePlaybackServiceRunning() bool {
	samplePlaybackService.mutex.Lock()
	defer samplePlaybackService.mutex.Unlock()
	return samplePlaybackService.service != nil && samplePlaybackService.service.Running()
}

func SetSamplePlaybackServiceCompressed(enabled bool) bool {
	return withSamplePlaybackService(func(service *ss.Service) {
		service.SetCompressed(enabled)
	})
}

func SetSamplePlaybackServiceWords(words int) bool {
	return withSamplePlaybackService(func(service *ss.Service) {
		service.SetDefaultWords(words)
	})
}

func SetSamplePlaybackServiceReverbWet(wet float64) bool {
	return withSamplePlaybackService(func(service *ss.Service) {
		service.SetReverbWet(wet)
	})
}

func SetSamplePlaybackServiceReverbLength(length float64) bool {
	return withSamplePlaybackService(func(service *ss.Service) {
		service.SetReverbLength(length)
	})
}

func ReloadSamplePlaybackServiceSamples() error {
	if !samplePlaybackProcessAllowed() {
		return fmt.Errorf("sample playback is disabled in %s mode", CurrentMode())
	}
	if !SamplePlaybackServiceRunning() {
		if err := StartSamplePlaybackService(); err != nil {
			return err
		}
	}
	var reloadErr error
	if !withSamplePlaybackService(func(service *ss.Service) {
		if IsBSSMode() {
			reloadErr = service.ReloadSigilSamples()
		} else {
			reloadErr = syncProSamplePlaybackSamples(service)
		}
		if reloadErr == nil {
			logSelectedSamplePlaybackFiles("ReloadSamplePlaybackServiceSamples", service)
		}
	}) {
		return fmt.Errorf("sample playback service is not running")
	}
	return reloadErr
}

func logSelectedSamplePlaybackFiles(context string, service *ss.Service) {
	if service == nil || service.State() == nil {
		return
	}
	for _, file := range service.State().SelectedSampleFiles() {
		LogInfo("SamplePlayback selected file", "context", context, "sigil", file.Sigil, "file", filepath.Base(file.Path), "path", file.Path)
	}
}

func withSamplePlaybackService(fn func(*ss.Service)) bool {
	if !samplePlaybackProcessAllowed() {
		return false
	}
	samplePlaybackService.mutex.Lock()
	service := samplePlaybackService.service
	samplePlaybackService.mutex.Unlock()
	if service == nil || !service.Running() {
		return false
	}
	fn(service)
	return true
}

func samplePlaybackProcessAllowed() bool {
	return IsBSSMode() || anyProSamplePlaybackEnabled()
}

func anyProSamplePlaybackEnabled() bool {
	if IsBSSMode() {
		return false
	}
	for _, patchName := range patchNames {
		if proSamplePlaybackEnabled(GetPatch(patchName)) {
			return true
		}
	}
	return false
}

func proSamplePlaybackEnabled(patch *Patch) bool {
	return patch != nil && !IsBSSMode() && patch.GetBool("sound.samplesplitter")
}

func firstProSamplePlaybackDir() (string, error) {
	for _, patchName := range patchNames {
		patch := GetPatch(patchName)
		if !proSamplePlaybackEnabled(patch) {
			continue
		}
		return proSamplePlaybackDirForPatch(patch)
	}
	return "", fmt.Errorf("no pro SampleSplitter patch is enabled")
}

func proSamplePlaybackDirForPatch(patch *Patch) (string, error) {
	if patch == nil {
		return "", fmt.Errorf("no patch")
	}
	name := strings.TrimSpace(patch.Get("sound.samplesplitterdir"))
	if name == "" {
		return "", fmt.Errorf("sound.samplesplitterdir is empty for patch %s", patch.Name())
	}
	cleanName := filepath.Clean(name)
	if filepath.IsAbs(cleanName) || cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("sound.samplesplitterdir must be a directory name under config/mp3")
	}
	return samplePlaybackConfigMP3Dir(cleanName)
}

func samplePlaybackConfigMP3Dir(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("sample playback mp3 directory name is empty")
	}
	cleanName := filepath.Clean(name)
	if filepath.IsAbs(cleanName) || cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("sample playback mp3 directory must be under config/mp3")
	}
	base, err := filepath.Abs(filepath.Join(ConfigDir(), "mp3"))
	if err != nil {
		return "", err
	}
	dir, err := filepath.Abs(filepath.Join(base, cleanName))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, dir)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("sample playback mp3 directory must stay under config/mp3")
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("samplesplitter mp3 directory not found: %s", dir)
	}
	return dir, nil
}

func SyncProSamplePlaybackServiceSamples() error {
	if IsBSSMode() {
		return nil
	}
	if !anyProSamplePlaybackEnabled() {
		StopSamplePlaybackService()
		return nil
	}
	if !SamplePlaybackServiceRunning() {
		return StartSamplePlaybackService()
	}
	var syncErr error
	if !withSamplePlaybackService(func(service *ss.Service) {
		syncErr = syncProSamplePlaybackSamples(service)
	}) {
		return fmt.Errorf("sample playback service is not running")
	}
	return syncErr
}

func EnsureProSamplePlaybackService() error {
	if IsBSSMode() {
		return nil
	}
	if !anyProSamplePlaybackEnabled() {
		return fmt.Errorf("no pro SampleSplitter patch is enabled")
	}
	if !SamplePlaybackServiceRunning() {
		return StartSamplePlaybackService()
	}
	return nil
}

func syncProSamplePlaybackSamples(service *ss.Service) error {
	if service == nil {
		return fmt.Errorf("sample playback service is not running")
	}
	for _, patchName := range patchNames {
		channel := SamplePlaybackChannelForPatch(patchName)
		patch := GetPatch(patchName)
		if !proSamplePlaybackEnabled(patch) {
			service.ClearChannelSample(channel)
			continue
		}
		dir, err := proSamplePlaybackDirForPatch(patch)
		if err != nil {
			return err
		}
		if err := service.LoadChannelSample(channel, dir); err != nil {
			return fmt.Errorf("load samplesplitter patch %s from %s: %w", patchName, dir, err)
		}
	}
	return nil
}

func IsSamplesplitterSynth(synth *Synth) bool {
	if synth == nil {
		return false
	}
	return strings.EqualFold(synth.portchannel.port, samplesplitterMidiPort)
}

func SendSamplesplitterNote(synth *Synth, value any) bool {
	if !IsSamplesplitterSynth(synth) {
		return false
	}
	if !IsBSSMode() {
		LogOfType("midi", "SendSamplesplitterNote: blocked by global.mode", "mode", CurrentMode())
		return true
	}
	channel := synth.portchannel.channel - 1
	if channel < 0 {
		channel = 0
	}
	return withSamplePlaybackService(func(service *ss.Service) {
		switch v := value.(type) {
		case *NoteOn:
			if err := service.NoteOn(channel, int(v.Pitch), int(v.Velocity)); err != nil {
				LogWarn("SendSamplesplitterNote NoteOn", "err", err)
			}
		case *NoteOff:
			service.NoteOff(channel, int(v.Pitch))
		default:
			LogWarn("SendSamplesplitterNote: doesn't handle", "type", fmt.Sprintf("%T", v))
		}
	})
}

func SendSamplesplitterPitchBend(synth *Synth, value int) bool {
	if !IsSamplesplitterSynth(synth) {
		return false
	}
	if !IsBSSMode() {
		LogOfType("midi", "SendSamplesplitterPitchBend: blocked by global.mode", "mode", CurrentMode())
		return true
	}
	channel := synth.portchannel.channel - 1
	if channel < 0 {
		channel = 0
	}
	return withSamplePlaybackService(func(service *ss.Service) {
		service.MIDIPitchBend(channel, value)
	})
}

func SamplesplitterProcessInfo() *ProcessInfo {
	return NewProcessInfo(SamplesplitterExe, "in-engine", "", nil)
}

func SamplesplitterRuntimeDir() string {
	exeDir := ""
	candidates := []string{}
	if currentExe, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(currentExe)
		candidates = append(candidates,
			filepath.Clean(filepath.Join(exeDir, "..", "samplesplitter")),
			filepath.Clean(filepath.Join(exeDir, "samplesplitter")),
			filepath.Clean(filepath.Join(exeDir, "..", "..", "pkg", "samplesplitter", "assets")),
			exeDir,
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "assets"),
			filepath.Join(cwd, "pkg", "samplesplitter", "assets"),
			filepath.Clean(filepath.Join(cwd, "..", "pkg", "samplesplitter", "assets")),
			filepath.Clean(filepath.Join(cwd, "..", "..", "pkg", "samplesplitter", "assets")),
			filepath.Join(cwd, "samplesplitter"),
		)
	}
	for _, candidate := range candidates {
		if FileExists(filepath.Join(candidate, "static", "index.html")) {
			return candidate
		}
	}
	return exeDir
}

func SamplesplitterMP3Dir(runtimeDir string) string {
	dir, err := samplePlaybackConfigMP3Dir("bss")
	if err != nil {
		LogWarn("SamplesplitterMP3Dir: config/mp3/bss unavailable", "err", err)
		return filepath.Join(ConfigDir(), "mp3", "bss")
	}
	return dir
}
