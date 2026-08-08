package samplesplitter

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
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

	Config         Config
	CurrentFile    string
	CueData        *CueData
	Waveform       []float64
	SigilSamples   map[string]SampleState
	ChannelSamples map[int]SampleState
	// ChannelRotation holds every analyzed sample in a channel's directory,
	// populated only when that channel has rotation enabled. PlanNoteOn picks
	// one at random per note instead of always using ChannelSamples.
	ChannelRotation map[int][]SampleState
	ChannelRotate   map[int]bool
	// ChannelMode and ChannelLoop are per-channel because the samplesplitter
	// and the sampleplayer can be active on different patches at once: the
	// splitter carves a file into looping splits, the player plays whole
	// files one-shot.
	ChannelMode map[int]string
	ChannelLoop map[int]bool
	// ChannelDir is the directory the channel's samples were loaded from,
	// used to tell a real settings change from a repeated reload request.
	ChannelDir        map[int]string
	rotateLast        map[int]int
	rotateRNG         *rand.Rand
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
	// MaxRMS is the analyzed loudness of the source file, carried through so
	// callers can tell a quiet sample from a loud one without re-analyzing.
	MaxRMS         float64 `json:"max_rms"`
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
	ChannelSamples     map[int]SampleState    `json:"channel_samples,omitempty"`
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
	ReverbWet          float64                `json:"reverb_wet"`
	ReverbLength       float64                `json:"reverb_length"`
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
		Config:          config,
		SigilSamples:    make(map[string]SampleState),
		ChannelSamples:  make(map[int]SampleState),
		ChannelRotation: make(map[int][]SampleState),
		ChannelRotate:   make(map[int]bool),
		ChannelMode:     make(map[int]string),
		ChannelLoop:     make(map[int]bool),
		ChannelDir:      make(map[int]string),
		rotateLast:      make(map[int]int),
		PitchBendSemis:  make(map[int]float64),
		AudioError:      "audio playback is not implemented in the Go port yet",
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
	channelSamples := make(map[int]SampleState, len(s.ChannelSamples))
	for channel, sample := range s.ChannelSamples {
		if sample.CurrentFile != "" {
			sample.CurrentFile = filepath.Base(sample.CurrentFile)
		}
		channelSamples[channel] = sample
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
		ChannelSamples:     channelSamples,
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
		ReverbWet:          s.Config.ReverbWet,
		ReverbLength:       s.Config.ReverbLength,
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
	for channel, sample := range s.ChannelSamples {
		if sample.CurrentFile == "" || sample.CueData == nil || seen[sample.CurrentFile] {
			continue
		}
		files = append(files, SelectedSampleFile{Sigil: fmt.Sprintf("channel-%d", channel), Path: sample.CurrentFile})
		seen[sample.CurrentFile] = true
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

func (s *State) SetReverbWet(wet float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.ReverbWet = clampReverbWet(wet)
}

func (s *State) SetReverbLength(length float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.ReverbLength = clampReverbLength(length)
}

func (s *State) SetDefaultWords(words int) {
	if words < 1 {
		words = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.DefaultWords = words
}

func (s *State) SetMinimumMP3Duration(seconds float64) {
	if seconds < 0 {
		seconds = DefaultMinimumMP3DurationSeconds
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.MinimumMP3DurationSeconds = seconds
}

func (s *State) SetWordThreshold(threshold float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.WordThreshold = clampWordThreshold(threshold)
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
	for _, sample := range s.ChannelSamples {
		add(sample.CurrentFile)
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
	if s.ChannelRotate[channel] {
		if picked, ok := s.randomRotationSampleLocked(channel); ok {
			sample = picked
		}
	}
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
	if splitIndex >= len(splits) {
		splitIndex = len(splits) - 1
	}
	if splitIndex < 0 {
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
		MaxRMS:         sample.CueData.MaxRMS,
		PitchSemitones: round4(semitones),
		PitchRatio:     round4(math.Pow(2.0, semitones/12.0)),
		Loop:           s.channelLoopLocked(channel),
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
	if sample, ok := s.ChannelSamples[channel]; ok && sample.CurrentFile != "" && sample.CueData != nil {
		return sample
	}
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

func (s *State) ClearChannelSample(channel int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ChannelSamples, channel)
	delete(s.ChannelRotation, channel)
	delete(s.ChannelRotate, channel)
	delete(s.ChannelMode, channel)
	delete(s.ChannelLoop, channel)
	delete(s.ChannelDir, channel)
	delete(s.rotateLast, channel)
}

// ChannelLoadedWith reports whether the channel already holds samples loaded
// with exactly these settings. Loading a preset re-applies every parameter, so
// the same channel gets asked to reload many times over; without this check
// each of those re-runs ffmpeg across the whole directory.
func (s *State) ChannelLoadedWith(channel int, dir, mode string, loop, rotate bool) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ChannelDir[channel] != dir ||
		s.ChannelMode[channel] != mode ||
		s.ChannelLoop[channel] != loop ||
		s.ChannelRotate[channel] != rotate {
		return false
	}
	// Settings match, but only skip if something is actually loaded.
	if rotate {
		return len(s.ChannelRotation[channel]) > 0
	}
	sample, ok := s.ChannelSamples[channel]
	return ok && sample.CurrentFile != "" && sample.CueData != nil
}

// SetChannelDir records the directory a channel's samples came from, so
// ChannelLoadedWith can tell a genuine change from a repeat request.
func (s *State) SetChannelDir(channel int, dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChannelDir == nil {
		s.ChannelDir = make(map[int]string)
	}
	s.ChannelDir[channel] = dir
}

// SetChannelPlayback records how a channel turns an MP3 into notes: mode is
// the analyze mode used when loading, loop is whether a triggered note repeats
// while it is held. The samplesplitter loops its splits; the sampleplayer
// plays a whole file once.
func (s *State) SetChannelPlayback(channel int, mode string, loop bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChannelMode == nil {
		s.ChannelMode = make(map[int]string)
	}
	if s.ChannelLoop == nil {
		s.ChannelLoop = make(map[int]bool)
	}
	s.ChannelMode[channel] = mode
	s.ChannelLoop[channel] = loop
}

// channelLoopLocked reports whether notes on this channel loop. Channels with
// no recorded setting keep the historical behaviour of looping.
func (s *State) channelLoopLocked(channel int) bool {
	if loop, ok := s.ChannelLoop[channel]; ok {
		return loop
	}
	return true
}

// analyzeOptionsForChannel is defaultAnalyzeOptions with the channel's mode
// applied, so one patch can be splitting while another plays whole files.
func (s *State) analyzeOptionsForChannel(channel int) AnalyzeOptions {
	opts := s.defaultAnalyzeOptions()
	s.mu.RLock()
	mode := s.ChannelMode[channel]
	s.mu.RUnlock()
	if mode != "" {
		opts.Mode = mode
	}
	return opts
}

// minimumDurationForChannel returns the shortest MP3 the channel will accept.
// The configured minimum exists so the splitter isn't handed a file too short
// to carve into words; the sampleplayer plays files whole, and short one-shots
// are exactly what it is for, so it has no minimum.
func (s *State) minimumDurationForChannel(channel int) float64 {
	s.mu.RLock()
	mode := s.ChannelMode[channel]
	minimum := s.Config.MinimumMP3DurationSeconds
	s.mu.RUnlock()
	if mode == WholeSplitMode {
		return 0
	}
	return minimum
}

// SetChannelRotate turns per-note random sample selection on or off for a
// channel. Turning it off drops the analyzed rotation set, so the channel
// falls back to the single sample loaded by LoadChannelDefault.
func (s *State) SetChannelRotate(channel int, rotate bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChannelRotate == nil {
		s.ChannelRotate = make(map[int]bool)
	}
	s.ChannelRotate[channel] = rotate
	if !rotate {
		delete(s.ChannelRotation, channel)
		delete(s.rotateLast, channel)
	}
}

// LoadChannelRotation analyzes every usable MP3 in dir and keeps them all, so
// PlanNoteOn can pick a different one for each note. It returns the paths that
// need preloading. Files that fail analysis are skipped rather than failing
// the whole load - one bad MP3 shouldn't silence the directory.
func (s *State) LoadChannelRotation(channel int, dir string, analyzer Analyzer) ([]string, error) {
	return s.loadChannelRotation(channel, dir, analyzer.AnalyzeFile)
}

func (s *State) loadChannelRotation(channel int, dir string, analyze analyzeMP3Func) ([]string, error) {
	files, err := ListMP3FilesWithMinimumDuration(dir, s.minimumDurationForChannel(channel))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no usable MP3 files in %s", dir)
	}

	opts := s.analyzeOptionsForChannel(channel)
	samples := make([]SampleState, 0, len(files))
	paths := make([]string, 0, len(files))
	var lastErr error
	for _, mp3 := range files {
		cue, waveform, err := analyze(mp3.Path, opts)
		if err != nil {
			lastErr = err
			continue
		}
		samples = append(samples, SampleState{
			Sigil:       fmt.Sprintf("channel-%d", channel),
			CurrentFile: mp3.Path,
			CueData:     &cue,
			Waveform:    waveform,
		})
		paths = append(paths, mp3.Path)
	}
	if len(samples) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("no analyzable MP3 files in %s: %w", dir, lastErr)
		}
		return nil, fmt.Errorf("no analyzable MP3 files in %s", dir)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChannelRotation == nil {
		s.ChannelRotation = make(map[int][]SampleState)
	}
	if s.ChannelSamples == nil {
		s.ChannelSamples = make(map[int]SampleState)
	}
	s.ChannelRotation[channel] = samples
	delete(s.rotateLast, channel)
	// Keep ChannelSamples populated so anything reading the channel's current
	// sample (status, waveform display) still sees something sensible.
	s.ChannelSamples[channel] = samples[0]
	return paths, nil
}

// ChannelSampleInfo describes one sample a channel can play, for logging.
type ChannelSampleInfo struct {
	Path      string
	MaxRMS    float64
	Duration  float64
	NumSplits int
}

// ChannelSampleInventory lists every sample the channel can currently play:
// the whole rotation set when rotation is on, otherwise the single loaded
// sample. MaxRMS makes it obvious which files are quiet.
func (s *State) ChannelSampleInventory(channel int) []ChannelSampleInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	samples := s.ChannelRotation[channel]
	if len(samples) == 0 {
		if sample, ok := s.ChannelSamples[channel]; ok {
			samples = []SampleState{sample}
		}
	}
	infos := make([]ChannelSampleInfo, 0, len(samples))
	for _, sample := range samples {
		info := ChannelSampleInfo{Path: sample.CurrentFile}
		if sample.CueData != nil {
			info.MaxRMS = sample.CueData.MaxRMS
			info.Duration = sample.CueData.Duration
			info.NumSplits = sample.CueData.NumSplits
		}
		infos = append(infos, info)
	}
	return infos
}

// randomRotationSampleLocked picks a random sample from the channel's rotation
// set, avoiding an immediate repeat when there is more than one to choose from.
// Callers must hold s.mu.
func (s *State) randomRotationSampleLocked(channel int) (SampleState, bool) {
	candidates := s.ChannelRotation[channel]
	if len(candidates) == 0 {
		return SampleState{}, false
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}
	if s.rotateRNG == nil {
		s.rotateRNG = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if s.rotateLast == nil {
		s.rotateLast = make(map[int]int)
	}
	last, seen := s.rotateLast[channel]
	index := s.rotateRNG.Intn(len(candidates))
	if seen && index == last {
		// Shift to a neighbour rather than re-rolling, which keeps the
		// selection uniform over the remaining candidates.
		index = (index + 1 + s.rotateRNG.Intn(len(candidates)-1)) % len(candidates)
	}
	s.rotateLast[channel] = index
	return candidates[index], true
}

func (s *State) LoadChannelDefault(channel int, dir string, analyzer Analyzer) error {
	return s.loadChannelDefault(channel, dir, analyzer.AnalyzeFile)
}

func (s *State) loadChannelDefault(channel int, dir string, analyze analyzeMP3Func) error {
	files, err := ListMP3FilesWithMinimumDuration(dir, s.minimumDurationForChannel(channel))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no usable MP3 files in %s", dir)
	}
	mp3, cue, waveform, err := analyzeFirstUsableMP3(files, analyze, s.analyzeOptionsForChannel(channel))
	sample := SampleState{Sigil: fmt.Sprintf("channel-%d", channel), CurrentFile: mp3.Path}
	if err != nil {
		sample.Error = err.Error()
	} else {
		sample.CueData = &cue
		sample.Waveform = waveform
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ChannelSamples == nil {
		s.ChannelSamples = make(map[int]SampleState)
	}
	s.ChannelSamples[channel] = sample
	return err
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

type analyzeMP3Func func(string, AnalyzeOptions) (CueData, []float64, error)

func (s *State) defaultAnalyzeOptions() AnalyzeOptions {
	s.mu.RLock()
	config := s.Config
	s.mu.RUnlock()
	return AnalyzeOptions{
		Mode:             DefaultSplitMode,
		Interval:         DefaultIntervalSeconds,
		WordsPerSplit:    config.DefaultWords,
		SilenceThreshold: config.SilenceThreshold,
		SilenceMinimum:   config.SilenceMinimum,
		WordThreshold:    config.WordThreshold,
	}
}

func analyzeFirstUsableMP3(files []MP3File, analyze analyzeMP3Func, opts AnalyzeOptions) (MP3File, CueData, []float64, error) {
	var quietErr error
	for _, mp3 := range files {
		cue, waveform, err := analyze(mp3.Path, opts)
		if err == nil {
			return mp3, cue, waveform, nil
		}
		if errors.Is(err, ErrBelowWordThreshold) {
			quietErr = err
			continue
		}
		return mp3, CueData{}, nil, err
	}
	if quietErr != nil {
		return MP3File{}, CueData{}, nil, quietErr
	}
	return MP3File{}, CueData{}, nil, errors.New("no usable MP3 files")
}

func prefixedMP3Candidates(files []MP3File, prefix, excludePath string, rng *rand.Rand) []MP3File {
	matches := make([]MP3File, 0)
	for _, file := range files {
		if strings.HasPrefix(strings.ToLower(file.Name), strings.ToLower(prefix)) {
			matches = append(matches, file)
		}
	}
	if rng != nil {
		rng.Shuffle(len(matches), func(i, j int) {
			matches[i], matches[j] = matches[j], matches[i]
		})
	}
	excludePath = normalizePathForCompare(excludePath)
	if excludePath == "" || len(matches) < 2 {
		return matches
	}
	ordered := make([]MP3File, 0, len(matches))
	var excluded []MP3File
	for _, file := range matches {
		if normalizePathForCompare(file.Path) == excludePath {
			excluded = append(excluded, file)
		} else {
			ordered = append(ordered, file)
		}
	}
	return append(ordered, excluded...)
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
	files, listErr := ListMP3FilesWithMinimumDuration(s.Config.MP3Dir, s.Config.MinimumMP3DurationSeconds)
	opts := s.defaultAnalyzeOptions()

	for _, sigil := range Sigils {
		if listErr != nil {
			loaded[sigil] = SampleState{Sigil: sigil, Error: listErr.Error()}
			continue
		}
		candidates := prefixedMP3Candidates(files, sigil, previous[sigil], rng)
		if len(candidates) == 0 {
			loaded[sigil] = SampleState{Sigil: sigil, Error: "No MP3 files start with '" + sigil + "'"}
			continue
		}
		mp3, cue, waveform, err := analyzeFirstUsableMP3(candidates, analyzer.AnalyzeFile, opts)
		sample := SampleState{Sigil: sigil, CurrentFile: mp3.Path}
		if err != nil {
			if errors.Is(err, ErrBelowWordThreshold) {
				sample.Error = fmt.Sprintf("No MP3 files starting with '%s' exceed word threshold %.4f", sigil, opts.WordThreshold)
			} else {
				sample.Error = err.Error()
			}
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
	s.CurrentFile = ""
	s.CueData = nil
	s.Waveform = nil
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

	files, err := ListMP3FilesWithMinimumDuration(s.Config.MP3Dir, s.Config.MinimumMP3DurationSeconds)
	if err != nil || len(files) == 0 {
		return
	}
	mp3, cue, waveform, err := analyzeFirstUsableMP3(files, analyzer.AnalyzeFile, s.defaultAnalyzeOptions())
	if err != nil {
		return
	}
	s.SetCurrent(mp3.Path, cue, waveform)
}
