package samplesplitter

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"sync"
	"time"
)

type SampleState struct {
	Sigil       string    `json:"sigil"`
	CurrentFile string    `json:"current_file,omitempty"`
	CueData     *CueData  `json:"cue_data,omitempty"`
	Waveform    []float64 `json:"-"`
	Error       string    `json:"error,omitempty"`
}

type State struct {
	mu sync.RWMutex

	Config            Config
	CurrentFile       string
	CueData           *CueData
	Waveform          []float64
	SigilSamples      map[string]SampleState
	MIDIPort          string
	MIDIError         string
	MIDIActivityCount int64
	MIDIActivityTime  *time.Time
	PitchBendSemis    map[int]float64
	LastPlayback      *PlaybackRequest
	ActiveVoices      []string
	AudioError        string
	Busy              bool
	BusyMessage       string
	PyoReady          bool
	AudioOutputID     *int
	AudioOutputName   *string
}

type PlaybackRequest struct {
	Type           string  `json:"type"`
	Sigil          string  `json:"sigil,omitempty"`
	File           string  `json:"file,omitempty"`
	FilePath       string  `json:"-"`
	VoiceKey       string  `json:"voice,omitempty"`
	Note           int     `json:"note"`
	Velocity       int     `json:"velocity"`
	Channel        int     `json:"channel"`
	SplitIndex     int     `json:"split_index"`
	StartSec       float64 `json:"start_sec"`
	EndSec         float64 `json:"end_sec"`
	PitchSemitones float64 `json:"pitch_semitones"`
	PitchRatio     float64 `json:"pitch_ratio"`
	Loop           bool    `json:"loop"`
	Compressed     bool    `json:"compressed"`
}

type StateSnapshot struct {
	CurrentFile        string                 `json:"current_file"`
	CueData            *CueData               `json:"cue_data"`
	Waveform           []float64              `json:"waveform"`
	SigilSamples       map[string]SampleState `json:"sigil_samples"`
	MIDIPort           string                 `json:"midi_port"`
	MIDIError          string                 `json:"midi_error"`
	MIDIActivityCount  int64                  `json:"midi_activity_count"`
	MIDIActivityTime   *time.Time             `json:"midi_activity_time"`
	LastPlayback       *PlaybackRequest       `json:"last_playback,omitempty"`
	BaseNote           int                    `json:"base_note"`
	PeakStartEnabled   bool                   `json:"peak_start_enabled"`
	PitchBendSemitones float64                `json:"pitch_bend_semitones"`
	ActiveVoices       []string               `json:"active_voices"`
	Compressed         bool                   `json:"compressed"`
	Busy               bool                   `json:"busy"`
	BusyMessage        string                 `json:"busy_message,omitempty"`
	PyoReady           bool                   `json:"pyo_ready"`
	AudioError         string                 `json:"audio_error"`
	AudioOutputID      *int                   `json:"audio_output_id"`
	AudioOutputName    *string                `json:"audio_output_name"`
}

type SelectedSampleFile struct {
	Sigil string
	Path  string
}

func NewState(config Config) *State {
	return &State{
		Config:         config,
		SigilSamples:   make(map[string]SampleState),
		PitchBendSemis: make(map[int]float64),
		AudioError:     "audio playback is not implemented in the Go port yet",
	}
}

func (s *State) Snapshot() StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sigilSamples := make(map[string]SampleState, len(s.SigilSamples))
	for sigil, sample := range s.SigilSamples {
		if sample.CurrentFile != "" {
			sample.CurrentFile = filepath.Base(sample.CurrentFile)
		}
		sigilSamples[sigil] = sample
	}

	current := ""
	if s.CurrentFile != "" {
		current = filepath.Base(s.CurrentFile)
	}

	return StateSnapshot{
		CurrentFile:        current,
		CueData:            s.CueData,
		Waveform:           s.Waveform,
		SigilSamples:       sigilSamples,
		MIDIPort:           s.MIDIPort,
		MIDIError:          s.MIDIError,
		MIDIActivityCount:  s.MIDIActivityCount,
		MIDIActivityTime:   s.MIDIActivityTime,
		LastPlayback:       s.LastPlayback,
		BaseNote:           s.Config.BaseNote,
		PeakStartEnabled:   s.Config.PeakStartEnabled,
		PitchBendSemitones: round4(s.PitchBendSemis[-1]),
		ActiveVoices:       append([]string(nil), s.ActiveVoices...),
		Compressed:         s.Config.Compressed,
		Busy:               s.Busy,
		BusyMessage:        s.BusyMessage,
		PyoReady:           s.PyoReady,
		AudioError:         s.AudioError,
		AudioOutputID:      s.AudioOutputID,
		AudioOutputName:    s.AudioOutputName,
	}
}

func (s *State) SelectedSampleFiles() []SelectedSampleFile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files := make([]SelectedSampleFile, 0, len(Sigils)+1)
	seen := map[string]bool{}
	for _, sigil := range Sigils {
		sample, ok := s.SigilSamples[sigil]
		if !ok || sample.CurrentFile == "" || sample.CueData == nil {
			continue
		}
		files = append(files, SelectedSampleFile{Sigil: sigil, Path: sample.CurrentFile})
		seen[sample.CurrentFile] = true
	}
	if s.CurrentFile != "" && !seen[s.CurrentFile] {
		files = append(files, SelectedSampleFile{Sigil: "current", Path: s.CurrentFile})
	}
	return files
}

func (s *State) SetBaseNote(note int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.BaseNote = note
}

func (s *State) SetPeakStart(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.PeakStartEnabled = enabled
}

func (s *State) SetCompressed(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.Compressed = enabled
}

func (s *State) SetDefaultWords(words int) {
	if words < 1 {
		words = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.DefaultWords = words
}

func (s *State) SetBusy(busy bool, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Busy = busy
	if busy {
		s.BusyMessage = message
	} else {
		s.BusyMessage = ""
	}
}

func (s *State) SetAudioStatus(ready bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PyoReady = ready
	if err != nil {
		s.AudioError = err.Error()
		return
	}
	s.AudioError = ""
}

func (s *State) SetAudioOutput(id *int, name *string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AudioOutputID = id
	s.AudioOutputName = name
}

func (s *State) SetActiveVoices(voices []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveVoices = append([]string(nil), voices...)
}

func (s *State) StartupSamplePaths() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	paths := make([]string, 0, len(s.SigilSamples)+1)
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		paths = append(paths, path)
	}
	for _, sigil := range Sigils {
		if sample, ok := s.SigilSamples[sigil]; ok {
			add(sample.CurrentFile)
		}
	}
	add(s.CurrentFile)
	return paths
}

func (s *State) SetMIDIStatus(portName string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.MIDIError = err.Error()
		return
	}
	s.MIDIPort = portName
	s.Config.MIDIPortName = portName
	s.MIDIError = ""
}

func (s *State) RecordMIDIActivity() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MIDIActivityCount++
	s.MIDIActivityTime = &now
}

func (s *State) SetPitchBend(channel int, bendValue int) {
	semitones := (float64(bendValue) / 8192.0) * 12.0
	s.SetPitchBendSemitones(channel, semitones)
}

func (s *State) SetPitchBendSemitones(channel int, semitones float64) {
	semitones = maxFloat(-12, minFloat(12, semitones))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PitchBendSemis[channel] = semitones
}

func (s *State) PlanNoteOn(note, velocity, channel int) (*PlaybackRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sample := s.sampleForChannelLocked(channel)
	if sample.CueData == nil || sample.CurrentFile == "" {
		return nil, fmt.Errorf("no sample loaded for MIDI channel %d", channel)
	}
	splits := sample.CueData.Splits
	if len(splits) == 0 {
		return nil, fmt.Errorf("sample has no splits")
	}

	splitIndex := note - s.Config.BaseNote
	if note < s.Config.BaseNote {
		splitIndex = int((float64(note) / math.Max(1, float64(s.Config.BaseNote))) * float64(len(splits)))
		splitIndex = min(len(splits)-1, max(0, splitIndex))
	}
	if splitIndex < 0 || splitIndex >= len(splits) {
		return nil, fmt.Errorf("split index %d out of range for %d splits", splitIndex, len(splits))
	}

	start := splits[splitIndex]
	end := sample.CueData.Duration
	if splitIndex+1 < len(splits) {
		end = splits[splitIndex+1]
	}
	if s.Config.PeakStartEnabled && splitIndex < len(sample.CueData.PeakStarts) {
		start = minFloat(maxFloat(start, sample.CueData.PeakStarts[splitIndex]), end)
	}

	semitones := s.PitchBendSemis[channel]
	request := &PlaybackRequest{
		Type:           "note_on",
		Sigil:          sample.Sigil,
		File:           filepath.Base(sample.CurrentFile),
		FilePath:       sample.CurrentFile,
		VoiceKey:       fmt.Sprintf("midi-%d-%d", channel, note),
		Note:           note,
		Velocity:       velocity,
		Channel:        channel,
		SplitIndex:     splitIndex,
		StartSec:       round4(start),
		EndSec:         round4(end),
		PitchSemitones: round4(semitones),
		PitchRatio:     round4(math.Pow(2.0, semitones/12.0)),
		Loop:           true,
		Compressed:     s.Config.Compressed,
	}
	s.LastPlayback = request
	return request, nil
}

func (s *State) PlanPreview(splitIndex int, voiceKey string, velocity int) (*PlaybackRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if voiceKey == "" {
		voiceKey = "preview"
	}
	if velocity <= 0 {
		velocity = 110
	}
	if s.CueData == nil || s.CurrentFile == "" {
		return nil, fmt.Errorf("no file has been analyzed")
	}
	splits := s.CueData.Splits
	if splitIndex < 0 || splitIndex >= len(splits) {
		return nil, fmt.Errorf("split index %d out of range for %d splits", splitIndex, len(splits))
	}

	start := splits[splitIndex]
	end := s.CueData.Duration
	if splitIndex+1 < len(splits) {
		end = splits[splitIndex+1]
	}
	if s.Config.PeakStartEnabled && splitIndex < len(s.CueData.PeakStarts) {
		start = minFloat(maxFloat(start, s.CueData.PeakStarts[splitIndex]), end)
	}

	semitones := s.PitchBendSemis[-1]
	request := &PlaybackRequest{
		Type:           "preview_on",
		File:           filepath.Base(s.CurrentFile),
		FilePath:       s.CurrentFile,
		VoiceKey:       voiceKey,
		Velocity:       velocity,
		Channel:        -1,
		SplitIndex:     splitIndex,
		StartSec:       round4(start),
		EndSec:         round4(end),
		PitchSemitones: round4(semitones),
		PitchRatio:     round4(math.Pow(2.0, semitones/12.0)),
		Compressed:     s.Config.Compressed,
	}
	s.LastPlayback = request
	return request, nil
}

func (s *State) PlanNoteOff(note, channel int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastPlayback = &PlaybackRequest{
		Type:    "note_off",
		Note:    note,
		Channel: channel,
	}
}

func (s *State) sampleForChannelLocked(channel int) SampleState {
	if sigil, ok := SigilByMIDIChannel[channel]; ok {
		if sample, ok := s.SigilSamples[sigil]; ok && sample.CurrentFile != "" && sample.CueData != nil {
			return sample
		}
	}
	return SampleState{
		CurrentFile: s.CurrentFile,
		CueData:     s.CueData,
		Waveform:    s.Waveform,
	}
}

func (s *State) SetCurrent(file string, cue CueData, waveform []float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentFile = file
	s.CueData = &cue
	s.Waveform = waveform
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (s *State) LoadSigilDefaults(analyzer Analyzer, rng *rand.Rand) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	previous := s.previousSigilFiles()
	loaded := make(map[string]SampleState)
	var first *SampleState

	for _, sigil := range Sigils {
		mp3, err := ChooseRandomPrefixedMP3Excluding(s.Config.MP3Dir, sigil, previous[sigil], rng)
		if err != nil {
			loaded[sigil] = SampleState{Sigil: sigil, Error: "No MP3 files start with '" + sigil + "'"}
			continue
		}
		cue, waveform, err := analyzer.AnalyzeFile(mp3.Path, AnalyzeOptions{
			Mode:             DefaultSplitMode,
			Interval:         DefaultIntervalSeconds,
			WordsPerSplit:    s.Config.DefaultWords,
			SilenceThreshold: s.Config.SilenceThreshold,
			SilenceMinimum:   s.Config.SilenceMinimum,
		})
		sample := SampleState{Sigil: sigil, CurrentFile: mp3.Path}
		if err != nil {
			sample.Error = err.Error()
		} else {
			sample.CueData = &cue
			sample.Waveform = waveform
			if first == nil {
				copy := sample
				first = &copy
			}
		}
		loaded[sigil] = sample
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.SigilSamples = loaded
	if first != nil {
		s.CurrentFile = first.CurrentFile
		s.CueData = first.CueData
		s.Waveform = first.Waveform
	}
}

func (s *State) previousSigilFiles() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	previous := make(map[string]string, len(Sigils))
	for _, sigil := range Sigils {
		if sample, ok := s.SigilSamples[sigil]; ok {
			previous[sigil] = sample.CurrentFile
		}
	}
	return previous
}

func (s *State) LoadFirstIfEmpty(analyzer Analyzer) {
	s.mu.RLock()
	hasCurrent := s.CurrentFile != ""
	s.mu.RUnlock()
	if hasCurrent {
		return
	}

	files, err := ListMP3Files(s.Config.MP3Dir)
	if err != nil || len(files) == 0 {
		return
	}
	cue, waveform, err := analyzer.AnalyzeFile(files[0].Path, AnalyzeOptions{
		Mode:             DefaultSplitMode,
		Interval:         DefaultIntervalSeconds,
		WordsPerSplit:    s.Config.DefaultWords,
		SilenceThreshold: s.Config.SilenceThreshold,
		SilenceMinimum:   s.Config.SilenceMinimum,
	})
	if err != nil {
		return
	}
	s.SetCurrent(files[0].Path, cue, waveform)
}
