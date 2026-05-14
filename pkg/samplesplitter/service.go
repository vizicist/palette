package samplesplitter

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ServiceOptions struct {
	Config       Config
	StaticDir    string
	EnableMIDI   bool
	EnableHTTP   bool
	SelectOutput bool
}

type Service struct {
	mu       sync.Mutex
	config   Config
	analyzer Analyzer
	state    *State
	audio    *AudioManager
	midi     *MIDIManager
	server   *http.Server
	listener net.Listener
	started  bool
}

func NewService(options ServiceOptions) (*Service, error) {
	config := options.Config
	if err := config.Normalize(); err != nil {
		return nil, err
	}
	if config.FFmpegPath == "" {
		config.FFmpegPath = FindFFmpeg("")
	}
	if config.MP3Dir == "" {
		return nil, errors.New("mp3 directory is required")
	}
	info, err := os.Stat(config.MP3Dir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("directory not found: %s", config.MP3Dir)
	}
	staticDir := options.StaticDir
	if staticDir == "" {
		staticDir = ResolveStaticDir("")
	}
	analyzer := Analyzer{FFmpegPath: config.FFmpegPath}
	state := NewState(config)
	state.LoadSigilDefaults(analyzer, rand.New(rand.NewSource(time.Now().UnixNano())))
	state.LoadFirstIfEmpty(analyzer)
	svc := &Service{
		config:   config,
		analyzer: analyzer,
		state:    state,
	}

	audio, err := NewAudioManager(config.FFmpegPath, state)
	if err != nil {
		state.SetAudioStatus(false, err)
	} else {
		svc.audio = audio
		state.SetAudioStatus(true, nil)
		if options.SelectOutput {
			if _, defaultID, err := audio.Devices(); err == nil {
				_, _ = audio.SetOutput(defaultID)
			}
		}
	}

	if options.EnableMIDI {
		midi, err := NewMIDIManager(state, audio)
		if err != nil {
			state.SetMIDIStatus("", err)
		} else {
			svc.midi = midi
		}
	}
	if options.EnableHTTP {
		server := Server{
			State:     state,
			Analyzer:  analyzer,
			StaticDir: staticDir,
			MIDI:      svc.midi,
			Audio:     svc.audio,
		}
		svc.server = &http.Server{
			Addr:    fmt.Sprintf("127.0.0.1:%d", config.Port),
			Handler: server.Handler(),
		}
	}
	return svc, nil
}

func (s *Service) Start(ctx context.Context) error {
	if s == nil {
		return errors.New("samplesplitter service is nil")
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	s.mu.Unlock()

	if s.midi != nil && s.config.MIDIPortName != "" {
		if _, err := s.midi.Start(s.config.MIDIPortName); err != nil && s.state != nil {
			s.state.SetMIDIStatus("", err)
		}
	}
	if s.server != nil {
		listener, err := net.Listen("tcp", s.server.Addr)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.listener = listener
		s.mu.Unlock()
		go func() {
			if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) && s.state != nil {
				s.state.SetAudioStatus(false, err)
			}
		}()
	}
	if ctx != nil {
		go func() {
			<-ctx.Done()
			_ = s.Close()
		}()
	}
	return nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	server := s.server
	s.server = nil
	listener := s.listener
	s.listener = nil
	midi := s.midi
	s.midi = nil
	audio := s.audio
	s.audio = nil
	s.started = false
	s.mu.Unlock()

	if listener != nil {
		_ = listener.Close()
	}
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}
	if midi != nil {
		_ = midi.Close()
	}
	if audio != nil {
		audio.Close()
	}
	return nil
}

func (s *Service) Running() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *Service) State() *State {
	if s == nil {
		return nil
	}
	return s.state
}

func (s *Service) NoteOn(channel, note, velocity int) error {
	if s == nil || s.state == nil {
		return errors.New("samplesplitter service is not initialized")
	}
	s.state.RecordMIDIActivity()
	if velocity <= 0 {
		s.state.PlanNoteOff(note, channel)
		if s.audio != nil {
			s.audio.StopNote(channel, note)
		}
		return nil
	}
	req, err := s.state.PlanNoteOn(note, velocity, channel)
	if err != nil {
		return err
	}
	if s.audio == nil {
		return errors.New("audio backend is not initialized")
	}
	if err := s.audio.Play(req); err != nil {
		s.state.SetAudioStatus(false, err)
		return err
	}
	s.state.SetAudioStatus(true, nil)
	return nil
}

func (s *Service) NoteOff(channel, note int) {
	if s == nil || s.state == nil {
		return
	}
	s.state.RecordMIDIActivity()
	s.state.PlanNoteOff(note, channel)
	if s.audio != nil {
		s.audio.StopNote(channel, note)
	}
}

func (s *Service) StopChannel(channel int) {
	if s == nil || s.state == nil {
		return
	}
	s.state.RecordMIDIActivity()
	s.state.PlanNoteOff(-1, channel)
	if s.audio != nil {
		s.audio.StopNote(channel, -1)
	}
}

func (s *Service) PitchBend(channel int, value int) {
	if s == nil || s.state == nil {
		return
	}
	s.state.RecordMIDIActivity()
	s.state.SetPitchBend(channel, value)
}

func (s *Service) MIDIPitchBend(channel int, value int) {
	if s == nil || s.state == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	if value > 16383 {
		value = 16383
	}
	semitones := (float64(value-8192) / 8192.0) * 12.0
	s.state.RecordMIDIActivity()
	s.state.SetPitchBendSemitones(channel, semitones)
}

func FindFFmpeg(baseDir string) string {
	name := "ffmpeg"
	if filepath.Separator == '\\' {
		name = "ffmpeg.exe"
	}
	candidates := []string{}
	if baseDir != "" {
		candidates = append(candidates, filepath.Join(baseDir, "ffmpeg", "bin", name))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "ffmpeg", "bin", name),
			filepath.Join(cwd, "cmd", "samplesplitter", "ffmpeg", "bin", name),
			filepath.Clean(filepath.Join(cwd, "..", "samplesplitter", "ffmpeg", "bin", name)),
		)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return name
}

func ResolveStaticDir(baseDir string) string {
	candidates := []string{}
	if baseDir != "" {
		candidates = append(candidates, filepath.Join(baseDir, "static"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "static"),
			filepath.Join(cwd, "cmd", "samplesplitter", "static"),
			filepath.Clean(filepath.Join(cwd, "..", "samplesplitter", "static")),
		)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
			return candidate
		}
	}
	return "static"
}
