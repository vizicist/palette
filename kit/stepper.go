package kit

import (
	"fmt"
	"math"
	"strconv"
	"sync"

	json "github.com/goccy/go-json"
)

const StepperNumSteps = 8
const StepperSamplesplitterVelocity = 110

var theStepper *Stepper

type Stepper struct {
	mutex             sync.RWMutex
	config            StepperConfig
	playing           bool
	lastStep          int
	lastPlayCycle     Clicks
	tracks            map[string]*StepperTrack
	recordedNoteOnMap map[string]*StepperEvent
	sampleVoices      *SampleVoiceLifecycle
	player            *StepperPlayer
}

type StepperTrack struct {
	Recording       bool
	Steps           [StepperNumSteps][]*StepperEvent
	lastRecordStep  int
	lastRecordCycle Clicks
}

type StepperEvent struct {
	Pitch      uint8   `json:"pitch"`
	Velocity   uint8   `json:"velocity"`
	Pressure   float64 `json:"pressure"`
	Duration   Clicks  `json:"duration"`
	Quant      Clicks  `json:"quant,omitempty"`
	SynthName  string  `json:"synth"`
	StartClick Clicks  `json:"-"`
}

type stepperStatus struct {
	Playing         bool                    `json:"playing"`
	Step            int                     `json:"step"`
	Click           Clicks                  `json:"click"`
	ClicksPerSecond Clicks                  `json:"clicks_per_second"`
	StepLength      Clicks                  `json:"step_length"`
	Tracks          map[string]stepperTrack `json:"tracks"`
}

type stepperTrack struct {
	Recording bool              `json:"recording"`
	Route     string            `json:"route"`
	Steps     [][]*StepperEvent `json:"steps"`
}

func NewStepper() *Stepper {
	config := NewStepperConfig()
	sampleVoices := NewSampleVoiceLifecycle()
	s := &Stepper{
		config:            config,
		playing:           false,
		lastStep:          -1,
		lastPlayCycle:     -1,
		tracks:            map[string]*StepperTrack{},
		recordedNoteOnMap: map[string]*StepperEvent{},
		sampleVoices:      sampleVoices,
	}
	s.player = NewStepperPlayer(config, sampleVoices, s.stepLength, s.pitchBendValue)
	for _, patch := range []string{"A", "B", "C", "D"} {
		s.tracks[patch] = &StepperTrack{
			Recording:       false,
			lastRecordStep:  -1,
			lastRecordCycle: -1,
		}
	}
	return s
}

func ExecuteStepperAPI(api string, apiargs map[string]string) (string, error) {
	if theStepper == nil {
		return "", fmt.Errorf("stepper is not initialized")
	}
	switch api {
	case "status":
		return theStepper.Status()
	case "play":
		theStepper.SetPlaying(true)
		return theStepper.Status()
	case "stop":
		theStepper.SetPlaying(false)
		return theStepper.Status()
	case "setrecord":
		patch, err := needStringArg("patch", api, apiargs)
		if err != nil {
			return "", err
		}
		onoff, err := needStringArg("onoff", api, apiargs)
		if err != nil {
			return "", err
		}
		return theStepper.SetRecording(patch, IsTrueValue(onoff))
	case "clear":
		patch, err := needStringArg("patch", api, apiargs)
		if err != nil {
			return "", err
		}
		return theStepper.ClearTrack(patch)
	case "toggle":
		patch, err := needStringArg("patch", api, apiargs)
		if err != nil {
			return "", err
		}
		stepStr, err := needStringArg("step", api, apiargs)
		if err != nil {
			return "", err
		}
		step, err := strconv.Atoi(stepStr)
		if err != nil {
			return "", fmt.Errorf("stepper.toggle: bad step value: %w", err)
		}
		return theStepper.ToggleStep(patch, step)
	case "setroute":
		patch, err := needStringArg("patch", api, apiargs)
		if err != nil {
			return "", err
		}
		route, err := needStringArg("route", api, apiargs)
		if err != nil {
			return "", err
		}
		return theStepper.SetRoute(patch, route)
	default:
		return "", fmt.Errorf("unrecognized stepper api=%s", api)
	}
}

func (s *Stepper) SetPlaying(playing bool) {
	playing = s.config.CoercePlaying(playing)
	s.mutex.Lock()
	s.playing = playing
	s.lastStep = -1
	s.lastPlayCycle = -1
	s.mutex.Unlock()
	if !playing {
		s.StopAllSamplesplitterVoices()
		s.ResetPitchBends()
	}
	NotifyStepperChanged()
}

func (s *Stepper) SetRecording(patch string, recording bool) (string, error) {
	track, err := s.trackForPatch(patch)
	if err != nil {
		return "", err
	}
	s.mutex.Lock()
	track.Recording = recording
	track.lastRecordStep = -1
	track.lastRecordCycle = -1
	if recording && s.config.SequencingEnabled() {
		s.playing = true
		s.lastStep = -1
		s.lastPlayCycle = -1
	}
	s.mutex.Unlock()
	status, err := s.Status()
	NotifyStepperChanged()
	return status, err
}

func (s *Stepper) SetAllRecording(recording bool) {
	s.mutex.Lock()
	for _, track := range s.tracks {
		track.Recording = recording
		track.lastRecordStep = -1
		track.lastRecordCycle = -1
	}
	if recording && s.config.SequencingEnabled() {
		s.playing = true
		s.lastStep = -1
		s.lastPlayCycle = -1
	}
	s.mutex.Unlock()
	NotifyStepperChanged()
}

func (s *Stepper) ClearTrack(patch string) (string, error) {
	track, err := s.trackForPatch(patch)
	if err != nil {
		return "", err
	}
	s.mutex.Lock()
	for i := range track.Steps {
		track.Steps[i] = nil
	}
	track.lastRecordStep = -1
	track.lastRecordCycle = -1
	for key := range s.recordedNoteOnMap {
		if len(key) > 0 && key[0:1] == patch {
			delete(s.recordedNoteOnMap, key)
		}
	}
	s.mutex.Unlock()
	s.StopSamplesplitterVoice(patch)
	status, err := s.Status()
	NotifyStepperChanged()
	return status, err
}

func (s *Stepper) ToggleStep(patch string, step int) (string, error) {
	track, err := s.trackForPatch(patch)
	if err != nil {
		return "", err
	}
	if step < 0 || step >= StepperNumSteps {
		return "", fmt.Errorf("stepper.toggle: step out of range: %d", step)
	}
	s.mutex.Lock()
	if len(track.Steps[step]) == 0 {
		track.Steps[step] = []*StepperEvent{{
			Pitch:    60,
			Velocity: 96,
			Pressure: 0.5,
			Duration: s.stepLength(),
			Quant:    s.stepLength(),
		}}
	} else {
		track.Steps[step] = nil
	}
	s.mutex.Unlock()
	status, err := s.Status()
	NotifyStepperChanged()
	return status, err
}

func (s *Stepper) SetRoute(patch string, route string) (string, error) {
	if err := s.config.SetRoute(patch, route); err != nil {
		return "", err
	}
	s.ResetPitchBends()
	status, err := s.Status()
	NotifyStepperChanged()
	return status, err
}

func (s *Stepper) Status() (string, error) {
	status, err := s.StatusSnapshot()
	if err != nil {
		return "", err
	}
	bytes, err := json.Marshal(status)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (s *Stepper) StatusSnapshot() (stepperStatus, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	currentClick := CurrentClick()
	status := stepperStatus{
		Playing:         s.playing,
		Step:            s.stepForClick(currentClick),
		Click:           currentClick,
		ClicksPerSecond: ClicksPerSecond(),
		StepLength:      s.stepLength(),
		Tracks:          map[string]stepperTrack{},
	}
	for _, patch := range []string{"A", "B", "C", "D"} {
		track := s.tracks[patch]
		steps := make([][]*StepperEvent, StepperNumSteps)
		for i := range track.Steps {
			steps[i] = append([]*StepperEvent{}, track.Steps[i]...)
		}
		status.Tracks[patch] = stepperTrack{
			Recording: track.Recording,
			Route:     string(s.config.RouteForPatch(patch)),
			Steps:     steps,
		}
	}
	return status, nil
}

func (s *Stepper) RecordNoteOn(patch string, note *NoteOn, pressure float64, atClick Clicks, quant Clicks) {
	if !s.config.SequencingEnabled() {
		return
	}
	if note == nil {
		return
	}
	track, err := s.trackForPatch(patch)
	if err != nil {
		return
	}
	step, cycle := s.nearestStep(atClick)

	s.mutex.Lock()
	if !track.Recording {
		s.mutex.Unlock()
		return
	}
	if !s.playing {
		s.playing = true
		s.lastStep = -1
		s.lastPlayCycle = -1
	}
	if track.lastRecordStep != step || track.lastRecordCycle != cycle {
		track.Steps[step] = nil
		track.lastRecordStep = step
		track.lastRecordCycle = cycle
	}
	event := &StepperEvent{
		Pitch:      note.Pitch,
		Velocity:   note.Velocity,
		Pressure:   boundValueZeroToOne(pressure),
		Duration:   s.stepLength(),
		Quant:      quant,
		SynthName:  synthName(note.Synth),
		StartClick: atClick,
	}
	track.Steps[step] = append(track.Steps[step], event)
	s.recordedNoteOnMap[s.recordKey(patch, note.Pitch, event.SynthName)] = event
	s.mutex.Unlock()
	NotifyStepperChanged()
}

func (s *Stepper) RecordNoteOff(patch string, note *NoteOff, atClick Clicks) {
	if !s.config.SequencingEnabled() {
		return
	}
	if note == nil {
		return
	}
	_, err := s.trackForPatch(patch)
	if err != nil {
		return
	}
	s.mutex.Lock()
	event, ok := s.recordedNoteOnMap[s.recordKey(patch, note.Pitch, synthName(note.Synth))]
	if !ok {
		s.mutex.Unlock()
		return
	}
	duration := atClick - event.StartClick
	if duration < 1 {
		duration = 1
	}
	event.Duration = duration
	delete(s.recordedNoteOnMap, s.recordKey(patch, note.Pitch, synthName(note.Synth)))
	s.mutex.Unlock()
	NotifyStepperChanged()
}

func (s *Stepper) AdvanceTo(click Clicks) {
	if !s.config.SequencingEnabled() {
		return
	}
	s.mutex.Lock()
	if !s.playing {
		s.mutex.Unlock()
		return
	}
	step, cycle := s.stepAt(click)
	if step == s.lastStep && cycle == s.lastPlayCycle {
		s.mutex.Unlock()
		return
	}
	s.lastStep = step
	s.lastPlayCycle = cycle

	eventsByPatch := map[string][]StepperEvent{}
	for _, patch := range []string{"A", "B", "C", "D"} {
		for _, event := range s.tracks[patch].Steps[step] {
			eventsByPatch[patch] = append(eventsByPatch[patch], *event)
		}
	}
	s.mutex.Unlock()

	for patch, events := range eventsByPatch {
		for _, event := range events {
			s.playEvent(patch, event, click)
		}
	}
}

func (s *Stepper) playEvent(patch string, event StepperEvent, atClick Clicks) {
	atClick = s.nextQuant(atClick, event.Quant)
	s.player.PlayEvent(patch, event, atClick)
}

func (s *Stepper) setActiveSamplesplitterVoice(patch string, synth *Synth, pitch uint8, velocity uint8) (*StepperSampleVoice, StepperSampleVoice) {
	return s.player.StartSamplesplitterVoice(patch, synth, pitch, velocity)
}

func (s *Stepper) SamplePlaybackStopIfCurrent(event *StepperSamplePlaybackStop) *NoteOff {
	return s.player.SamplePlaybackStopIfCurrent(event)
}

func (s *Stepper) StopSamplesplitterVoice(patch string) {
	s.player.StopSamplesplitterVoice(patch)
}

func (s *Stepper) StopAllSamplesplitterVoices() {
	s.player.StopAllSamplesplitterVoices()
}

func (s *Stepper) ResetPitchBends() {
	s.player.ResetPitchBends()
}

func (s *Stepper) CurrentStep() int {
	return s.stepForClick(CurrentClick())
}

func (s *Stepper) stepForClick(click Clicks) int {
	step, _ := s.stepAt(click)
	return step
}

func (s *Stepper) stepAt(click Clicks) (int, Clicks) {
	stepLen := s.stepLength()
	if stepLen < 1 {
		stepLen = 1
	}
	stepIndex := click / stepLen
	step := int(stepIndex % StepperNumSteps)
	cycle := stepIndex / StepperNumSteps
	return step, cycle
}

func (s *Stepper) nearestStep(click Clicks) (int, Clicks) {
	stepLen := s.stepLength()
	if stepLen < 1 {
		stepLen = 1
	}
	stepIndex := (click + stepLen/2) / stepLen
	step := int(stepIndex % StepperNumSteps)
	cycle := stepIndex / StepperNumSteps
	return step, cycle
}

func (s *Stepper) nextQuant(click Clicks, quant Clicks) Clicks {
	if quant <= 1 {
		return click
	}
	rem := click % quant
	quantized := click
	if (rem * 2) > quant {
		quantized += quant - rem
	} else {
		quantized -= rem
	}
	if quantized < click {
		quantized += quant
	}
	return quantized
}

func (s *Stepper) stepLength() Clicks {
	beats, err := GetParamInt("global.looping_beats")
	if err != nil || beats < 1 {
		beats = 8
	}
	factor := TempoFactor
	if factor <= 0 {
		factor = 1
	}
	loopLen := Clicks(float64(OneBeat*Clicks(beats)) / factor)
	stepLen := Clicks(math.Max(1, float64(loopLen)/float64(StepperNumSteps)))
	return stepLen
}

func (s *Stepper) pitchBendValue(pressure float64) int {
	p := boundValueZeroToOne(pressure)
	return int(math.Round(p * 16383.0))
}

func (s *Stepper) trackForPatch(patch string) (*StepperTrack, error) {
	if patch != "A" && patch != "B" && patch != "C" && patch != "D" {
		return nil, fmt.Errorf("bad stepper patch=%s", patch)
	}
	track, ok := s.tracks[patch]
	if !ok {
		return nil, fmt.Errorf("stepper track missing for patch=%s", patch)
	}
	return track, nil
}

func (s *Stepper) recordKey(patch string, pitch uint8, synth string) string {
	return fmt.Sprintf("%s:%s:%d", patch, synth, pitch)
}

func synthName(synth *Synth) string {
	if synth == nil {
		return ""
	}
	return synth.name
}
