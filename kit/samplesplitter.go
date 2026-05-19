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

var inEngineSamplesplitter struct {
	mutex   sync.Mutex
	service *ss.Service
	cancel  context.CancelFunc
}

func StartInEngineSamplesplitter() error {
	inEngineSamplesplitter.mutex.Lock()
	defer inEngineSamplesplitter.mutex.Unlock()
	if inEngineSamplesplitter.service != nil && inEngineSamplesplitter.service.Running() {
		return nil
	}
	runtimeDir := SamplesplitterRuntimeDir(SamplesplitterExecutablePath())
	config := ss.DefaultConfig()
	config.Port = samplesplitterPort
	config.MP3Dir = SamplesplitterMP3Dir(runtimeDir)
	config.MIDIPortName = ""
	config.FFmpegPath = ss.FindFFmpeg(runtimeDir)
	config.Compressed = samplePlaybackCompressed()
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
	inEngineSamplesplitter.service = service
	inEngineSamplesplitter.cancel = cancel
	LogInfo("StartInEngineSamplesplitter", "mp3Dir", config.MP3Dir, "ffmpeg", config.FFmpegPath)
	return nil
}

func samplePlaybackCompressed() bool {
	enabled, err := GetParamBool("global.sampleplaybackcompressed")
	if err == nil {
		return enabled
	}
	enabled, err = GetParamBool("global.transmissioncompressed")
	if err == nil {
		return enabled
	}
	LogIfError(err)
	return false
}

func samplePlaybackWords() (int, error) {
	words, err := GetParamInt("global.sampleplaybackwords")
	if err == nil {
		return words, nil
	}
	words, legacyErr := GetParamInt("global.transmissionwords")
	if legacyErr == nil {
		return words, nil
	}
	return 2, err
}

func StopInEngineSamplesplitter() {
	inEngineSamplesplitter.mutex.Lock()
	service := inEngineSamplesplitter.service
	cancel := inEngineSamplesplitter.cancel
	inEngineSamplesplitter.service = nil
	inEngineSamplesplitter.cancel = nil
	inEngineSamplesplitter.mutex.Unlock()
	if cancel != nil {
		cancel()
	}
	if service != nil {
		_ = service.Close()
	}
}

func InEngineSamplesplitterRunning() bool {
	inEngineSamplesplitter.mutex.Lock()
	defer inEngineSamplesplitter.mutex.Unlock()
	return inEngineSamplesplitter.service != nil && inEngineSamplesplitter.service.Running()
}

func SetInEngineSamplesplitterCompressed(enabled bool) bool {
	return withInEngineSamplesplitter(func(service *ss.Service) {
		service.SetCompressed(enabled)
	})
}

func SetInEngineSamplesplitterWords(words int) bool {
	return withInEngineSamplesplitter(func(service *ss.Service) {
		service.SetDefaultWords(words)
	})
}

func ReloadInEngineSamplesplitterSamples() error {
	if !InEngineSamplesplitterRunning() {
		if err := StartInEngineSamplesplitter(); err != nil {
			return err
		}
	}
	var reloadErr error
	if !withInEngineSamplesplitter(func(service *ss.Service) {
		reloadErr = service.ReloadSigilSamples()
	}) {
		return fmt.Errorf("in-engine samplesplitter is not running")
	}
	return reloadErr
}

func withInEngineSamplesplitter(fn func(*ss.Service)) bool {
	inEngineSamplesplitter.mutex.Lock()
	service := inEngineSamplesplitter.service
	inEngineSamplesplitter.mutex.Unlock()
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
	channel := synth.portchannel.channel - 1
	if channel < 0 {
		channel = 0
	}
	return withInEngineSamplesplitter(func(service *ss.Service) {
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
	channel := synth.portchannel.channel - 1
	if channel < 0 {
		channel = 0
	}
	return withInEngineSamplesplitter(func(service *ss.Service) {
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
	candidates := []string{
		filepath.Join(runtimeDir, "mp3s"),
		filepath.Clean(filepath.Join(runtimeDir, "..", "data_default", "samplesplitter", "mp3s")),
		filepath.Clean(filepath.Join(runtimeDir, "..", "..", "data_default", "samplesplitter", "mp3s")),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "data_default", "samplesplitter", "mp3s"),
			filepath.Clean(filepath.Join(cwd, "..", "data_default", "samplesplitter", "mp3s")),
			filepath.Clean(filepath.Join(cwd, "..", "..", "data_default", "samplesplitter", "mp3s")),
		)
	}
	for _, candidate := range candidates {
		if FileExists(candidate) {
			return candidate
		}
	}
	return filepath.Join(runtimeDir, "mp3s")
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
