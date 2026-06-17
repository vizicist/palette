package kit

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	ss "github.com/vizicist/palette/pkg/samplesplitter"
)

const samplesplitterPort = 9876
const samplesplitterMidiPort = "16. Internal MIDI"
const samplesplitterStatusTimeout = 500 * time.Millisecond
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
	if !IsBSSMode() {
		return fmt.Errorf("sample playback is disabled in %s mode", CurrentMode())
	}
	samplePlaybackService.mutex.Lock()
	defer samplePlaybackService.mutex.Unlock()
	if samplePlaybackService.service != nil && samplePlaybackService.service.Running() {
		return nil
	}
	runtimeDir := SamplesplitterRuntimeDir(SamplesplitterExecutablePath())
	config := ss.DefaultConfig()
	config.Port = samplesplitterPort
	config.MP3Dir = SamplesplitterMP3Dir(runtimeDir)
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
	if !IsBSSMode() {
		return fmt.Errorf("sample playback is disabled in %s mode", CurrentMode())
	}
	if !SamplePlaybackServiceRunning() {
		if err := StartSamplePlaybackService(); err != nil {
			return err
		}
	}
	var reloadErr error
	if !withSamplePlaybackService(func(service *ss.Service) {
		reloadErr = service.ReloadSigilSamples()
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
	if !IsBSSMode() {
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
	exe := SamplesplitterExecutablePath()
	if exe == "" {
		return NewProcessInfo(SamplesplitterExe, "in-engine", "", nil)
	}

	dir := SamplesplitterRuntimeDir(exe)
	mp3Dir := SamplesplitterMP3Dir(dir)
	arg := fmt.Sprintf("--dir %q --port %d --midi-port %q --no-open", mp3Dir, samplesplitterPort, samplesplitterMidiPort)
	pi := NewProcessInfo(filepath.Base(exe), exe, arg, nil)
	pi.DirPath = dir
	return pi
}

func SamplesplitterExecutablePath() string {
	candidates := []string{}
	if currentExe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(currentExe)
		candidates = append(candidates,
			filepath.Join(exeDir, SamplesplitterExe),
			filepath.Clean(filepath.Join(exeDir, "..", "bin", SamplesplitterExe)),
		)
	}
	if paletteDir := PaletteDir(); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "bin", SamplesplitterExe))
	}
	if paletteDir := os.Getenv("PALETTE"); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "bin", SamplesplitterExe))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "cmd", "samplesplitter", SamplesplitterExe),
			filepath.Join(cwd, SamplesplitterExe),
			filepath.Clean(filepath.Join(cwd, "..", "samplesplitter", SamplesplitterExe)),
			filepath.Clean(filepath.Join(cwd, "..", SamplesplitterExe)),
			filepath.Clean(filepath.Join(cwd, "..", "..", "samplesplitter", SamplesplitterExe)),
			filepath.Clean(filepath.Join(cwd, "..", "..", SamplesplitterExe)),
		)
	}
	for _, candidate := range candidates {
		if FileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func SamplesplitterRuntimeDir(exe string) string {
	exeDir := filepath.Dir(exe)
	if exe == "" {
		if currentExe, err := os.Executable(); err == nil {
			exeDir = filepath.Dir(currentExe)
		}
	}
	candidates := []string{
		filepath.Clean(filepath.Join(exeDir, "..", "samplesplitter")),
		filepath.Clean(filepath.Join(exeDir, "..", "cmd", "samplesplitter")),
		exeDir,
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "cmd", "samplesplitter"),
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
	return ss.DefaultMP3Dir()
}

func SamplesplitterScriptPath() string {
	candidates := []string{}
	if currentExe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(currentExe)
		candidates = append(candidates,
			filepath.Clean(filepath.Join(exeDir, "..", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(exeDir, "..", "..", "samplesplitter", "samplesplitter.py")),
		)
	}
	if paletteDir := PaletteDir(); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "samplesplitter", "samplesplitter.py"))
	}
	if paletteDir := os.Getenv("PALETTE"); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "samplesplitter", "samplesplitter.py"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "cmd", "samplesplitter", "samplesplitter.py"),
			filepath.Join(cwd, "samplesplitter", "samplesplitter.py"),
			filepath.Clean(filepath.Join(cwd, "..", "cmd", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(cwd, "..", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(cwd, "..", "..", "cmd", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(cwd, "..", "..", "samplesplitter", "samplesplitter.py")),
		)
	}
	for _, candidate := range candidates {
		if FileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func samplesplitterWebIsListening() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", samplesplitterPort), samplesplitterStatusTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
