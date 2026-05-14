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

type SamplesplitterNoteOn struct {
	Patch     string
	Channel   int
	Note      int
	Velocity  int
	PitchBend int
}

type SamplesplitterNoteOff struct {
	Patch   string
	Channel int
	Note    int
}

type SamplesplitterPitchBend struct {
	Patch   string
	Channel int
	Value   int
}

func SamplesplitterChannelForPatch(patch string) int {
	switch patch {
	case "A":
		return 0
	case "B":
		return 1
	case "C":
		return 2
	case "D":
		return 3
	default:
		return 0
	}
}

func (event *SamplesplitterNoteOn) Trigger() {
	if event == nil {
		return
	}
	if !withInEngineSamplesplitter(func(service *ss.Service) {
		LogInfo("SamplesplitterNoteOn.Trigger", "patch", event.Patch, "channel", event.Channel, "note", event.Note, "velocity", event.Velocity, "pitchbend", event.PitchBend)
		service.MIDIPitchBend(event.Channel, event.PitchBend)
		if err := service.NoteOn(event.Channel, event.Note, event.Velocity); err != nil {
			LogWarn("SamplesplitterNoteOn", "err", err, "patch", event.Patch, "channel", event.Channel, "note", event.Note)
		}
	}) {
		LogWarn("SamplesplitterNoteOn: in-engine service is not running", "patch", event.Patch)
	}
}

func (event *SamplesplitterNoteOff) Trigger() {
	if event == nil {
		return
	}
	if !withInEngineSamplesplitter(func(service *ss.Service) {
		if event.Note < 0 {
			LogInfo("SamplesplitterNoteOff.Trigger StopChannel", "patch", event.Patch, "channel", event.Channel)
			service.StopChannel(event.Channel)
			return
		}
		LogInfo("SamplesplitterNoteOff.Trigger", "patch", event.Patch, "channel", event.Channel, "note", event.Note)
		service.NoteOff(event.Channel, event.Note)
	}) {
		LogWarn("SamplesplitterNoteOff: in-engine service is not running", "patch", event.Patch)
	}
}

func (event *SamplesplitterPitchBend) Trigger() {
	if event == nil {
		return
	}
	if !withInEngineSamplesplitter(func(service *ss.Service) {
		LogInfo("SamplesplitterPitchBend.Trigger", "patch", event.Patch, "channel", event.Channel, "value", event.Value)
		service.MIDIPitchBend(event.Channel, event.Value)
	}) {
		LogWarn("SamplesplitterPitchBend: in-engine service is not running", "patch", event.Patch)
	}
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
