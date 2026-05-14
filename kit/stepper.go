package kit

import (
	"fmt"
	"math"
	"strconv"
	"sync"

	json "github.com/goccy/go-json"
)

const StepperNumSteps = 16
const StepperDefaultSamplesplitterSynth = "P_16_C_01"
const StepperSamplesplitterVelocity = 110

var theStepper *Stepper

type Stepper struct {
	mutex             sync.RWMutex
	playing           bool
	lastStep          int
	lastPlayCycle     Clicks
	tracks            map[string]*StepperTrack
	recordedNoteOnMap map[string]*StepperEvent
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
	SynthName  string  `json:"synth"`
	StartClick Clicks  `json:"-"`
}

type stepperStatus struct {
	Playing bool                    `json:"playing"`
	Step    int                     `json:"step"`
	Tracks  map[string]stepperTrack `json:"tracks"`
}

type stepperTrack struct {
	Recording bool              `json:"recording"`
	Route     string            `json:"route"`
	Steps     [][]*StepperEvent `json:"steps"`
}

func NewStepper() *Stepper {
	s := &Stepper{
		playing:           false,
		lastStep:          -1,
		lastPlayCycle:     -1,
		tracks:            map[string]*StepperTrack{},
		recordedNoteOnMap: map[string]*StepperEvent{},
	}
	for _, patch := range []string{"A", "B", "C", "D"} {
		s.tracks[patch] = &StepperTrack{
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
	case "status", "get":
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
	s.mutex.Lock()
	s.playing = playing
	s.lastStep = -1
	s.lastPlayCycle = -1
	s.mutex.Unlock()
	if !playing {
		s.ResetPitchBends()
	}
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
	if recording {
		s.playing = true
		s.lastStep = -1
		s.lastPlayCycle = -1
	}
	s.mutex.Unlock()
	return s.Status()
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
	return s.Status()
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
		}}
	} else {
		track.Steps[step] = nil
	}
	s.mutex.Unlock()
	return s.Status()
}

func (s *Stepper) SetRoute(patch string, route string) (string, error) {
	if !validStepperRoute(route) {
		return "", fmt.Errorf("stepper.setroute: bad route=%s", route)
	}
	p := GetPatch(patch)
	if p == nil {
		return "", fmt.Errorf("no such patch: %s", patch)
	}
	err := p.SetParam("stepper.route", route)
	if err != nil {
		return "", err
	}
	err = p.SaveQuadAndAlert()
	if err != nil {
		return "", err
	}
	s.ResetPitchBends()
	return s.Status()
}

func validStepperRoute(route string) bool {
	switch route {
	case "off", "bidule", "samplesplitter", "both":
		return true
	default:
		return false
	}
}

func (s *Stepper) Status() (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	status := stepperStatus{
		Playing: s.playing,
		Step:    s.CurrentStep(),
		Tracks:  map[string]stepperTrack{},
	}
	for _, patch := range []string{"A", "B", "C", "D"} {
		track := s.tracks[patch]
		steps := make([][]*StepperEvent, StepperNumSteps)
		for i := range track.Steps {
			steps[i] = append([]*StepperEvent{}, track.Steps[i]...)
		}
		status.Tracks[patch] = stepperTrack{
			Recording: track.Recording,
			Route:     s.routeForPatch(patch),
			Steps:     steps,
		}
	}
	bytes, err := json.Marshal(status)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (s *Stepper) RecordNoteOn(patch string, note *NoteOn, pressure float64, atClick Clicks) {
	if note == nil {
		return
	}
	track, err := s.trackForPatch(patch)
	if err != nil {
		return
	}
	step, cycle := s.nearestStep(atClick)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !track.Recording {
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
		SynthName:  synthName(note.Synth),
		StartClick: atClick,
	}
	track.Steps[step] = append(track.Steps[step], event)
	s.recordedNoteOnMap[s.recordKey(patch, note.Pitch, event.SynthName)] = event
}

func (s *Stepper) RecordNoteOff(patch string, note *NoteOff, atClick Clicks) {
	if note == nil {
		return
	}
	_, err := s.trackForPatch(patch)
	if err != nil {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	event, ok := s.recordedNoteOnMap[s.recordKey(patch, note.Pitch, synthName(note.Synth))]
	if !ok {
		return
	}
	duration := atClick - event.StartClick
	if duration < 1 {
		duration = 1
	}
	event.Duration = duration
	delete(s.recordedNoteOnMap, s.recordKey(patch, note.Pitch, synthName(note.Synth)))
}

func (s *Stepper) AdvanceTo(click Clicks) {
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
	route := s.routeForPatch(patch)
	if route == "bidule" || route == "both" {
		synth := s.biduleSynthForPatch(patch, event)
		if synth != nil {
			s.playTimedEvent(synth, patch, event, atClick)
		}
	}
	if route == "samplesplitter" || route == "both" {
		synth := s.samplesplitterSynthForPatch(patch)
		if synth != nil {
			s.playSamplesplitterEvent(synth, patch, event, atClick)
		}
	}
}

func (s *Stepper) playTimedEvent(synth *Synth, patch string, event StepperEvent, atClick Clicks) {
	if synth == nil {
		return
	}
	noteOn := NewNoteOn(synth, event.Pitch, event.Velocity)
	noteOff := NewNoteOff(synth, event.Pitch, event.Velocity)
	ScheduleAt(atClick, patch, NewPitchBend(synth, s.pitchBendValue(event.Pressure)))
	ScheduleAt(atClick, patch, noteOn)
	duration := event.Duration
	if duration < 1 {
		duration = s.stepLength()
	}
	ScheduleAt(atClick+duration, patch, noteOff)
	ScheduleAt(atClick+duration+1, patch, NewPitchBend(synth, MidiPitchBendCenter))
}

func (s *Stepper) playSamplesplitterEvent(synth *Synth, patch string, event StepperEvent, atClick Clicks) {
	if synth == nil {
		return
	}
	velocity := event.Velocity
	if velocity < StepperSamplesplitterVelocity {
		velocity = StepperSamplesplitterVelocity
	}
	noteOn := NewNoteOn(synth, event.Pitch, velocity)
	noteOff := NewNoteOff(synth, event.Pitch, velocity)
	ScheduleAt(atClick, patch, NewPitchBend(synth, s.pitchBendValue(event.Pressure)))
	ScheduleAt(atClick, patch, noteOn)
	ScheduleAt(atClick+1, patch, NewPitchBend(synth, MidiPitchBendCenter))
	duration := event.Duration
	if duration < 1 {
		duration = s.stepLength()
	}
	ScheduleAt(atClick+duration, patch, noteOff)
}

func (s *Stepper) ResetPitchBends() {
	synths := map[*Synth]bool{}
	for _, patch := range []string{"A", "B", "C", "D"} {
		p := GetPatch(patch)
		if p != nil {
			if synth := p.Synth(); synth != nil {
				synths[synth] = true
			}
		}
		if synth := s.samplesplitterSynthForPatch(patch); synth != nil {
			synths[synth] = true
		}
	}
	if synth := GetSynth(StepperDefaultSamplesplitterSynth); synth != nil {
		synths[synth] = true
	}
	for synth := range synths {
		synth.SendPitchBend(MidiPitchBendCenter)
	}
}

func (s *Stepper) biduleSynthForPatch(patch string, event StepperEvent) *Synth {
	p := GetPatch(patch)
	if p != nil {
		return p.Synth()
	}
	if event.SynthName != "" {
		return GetSynth(event.SynthName)
	}
	return nil
}

func (s *Stepper) samplesplitterSynthForPatch(patch string) *Synth {
	p := GetPatch(patch)
	if p == nil {
		return nil
	}
	synthName := p.Get("stepper.samplesplitter_synth")
	if synthName == "" {
		synthName = StepperDefaultSamplesplitterSynth
	}
	return GetSynth(synthName)
}

func (s *Stepper) routeForPatch(patch string) string {
	p := GetPatch(patch)
	if p == nil {
		return "off"
	}
	route := p.Get("stepper.route")
	if !validStepperRoute(route) {
		return "bidule"
	}
	return route
}

func (s *Stepper) CurrentStep() int {
	step, _ := s.stepAt(CurrentClick())
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

func (s *Stepper) stepLength() Clicks {
	beats, err := GetParamInt("global.looping_beats")
	if err != nil || beats < 1 {
		beats = 8
	}
	loopLen := OneBeat * Clicks(beats)
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
