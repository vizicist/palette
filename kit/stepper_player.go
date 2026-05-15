package kit

type StepperPlayer struct {
	config       StepperConfig
	sampleVoices *SampleVoiceLifecycle
	stepLength   func() Clicks
	pitchBend    func(float64) int
}

func NewStepperPlayer(config StepperConfig, sampleVoices *SampleVoiceLifecycle, stepLength func() Clicks, pitchBend func(float64) int) *StepperPlayer {
	return &StepperPlayer{
		config:       config,
		sampleVoices: sampleVoices,
		stepLength:   stepLength,
		pitchBend:    pitchBend,
	}
}

func (p *StepperPlayer) PlayEvent(patch string, event StepperEvent, atClick Clicks) {
	if p.config.RouteIncludesBidule(patch) {
		synth := p.config.BiduleSynthForPatch(patch, event)
		if synth != nil {
			p.playTimedEvent(synth, patch, event, atClick)
		}
	}
	if p.config.RouteIncludesSamples(patch) {
		synth := p.config.SamplesplitterSynthForPatch(patch)
		if synth != nil {
			p.playSamplesplitterEvent(synth, patch, event, atClick)
		}
	}
}

func (p *StepperPlayer) playTimedEvent(synth *Synth, patch string, event StepperEvent, atClick Clicks) {
	if synth == nil {
		return
	}
	noteOn := NewNoteOn(synth, event.Pitch, event.Velocity)
	noteOff := NewNoteOff(synth, event.Pitch, event.Velocity)
	ScheduleAt(atClick, patch, NewPitchBend(synth, p.pitchBend(event.Pressure)))
	ScheduleAt(atClick, patch, noteOn)
	duration := event.Duration
	if duration < 1 {
		duration = p.stepLength()
	}
	ScheduleAt(atClick+duration, patch, noteOff)
	ScheduleAt(atClick+duration+1, patch, NewPitchBend(synth, MidiPitchBendCenter))
}

func (p *StepperPlayer) playSamplesplitterEvent(synth *Synth, patch string, event StepperEvent, atClick Clicks) {
	if synth == nil {
		return
	}
	velocity := transmissionVelocityFromPressure(GetPatch(patch), event.Pressure)
	noteOn := NewNoteOn(synth, event.Pitch, velocity)
	previous, current := p.StartSamplesplitterVoice(patch, synth, event.Pitch, velocity)
	if previous != nil {
		if noteOff := previous.NoteOff(); noteOff != nil {
			ScheduleAt(atClick, patch, noteOff)
		}
	}
	ScheduleAt(atClick, patch, NewPitchBend(synth, p.pitchBend(event.Pressure)))
	ScheduleAt(atClick, patch, noteOn)
	ScheduleAt(atClick+1, patch, NewPitchBend(synth, MidiPitchBendCenter))
	duration := event.Duration
	if duration < 1 {
		duration = p.stepLength()
	}
	ScheduleAt(atClick+duration, patch, &StepperSamplePlaybackStop{Voice: current})
}

func (p *StepperPlayer) StartSamplesplitterVoice(patch string, synth *Synth, pitch uint8, velocity uint8) (*StepperSampleVoice, StepperSampleVoice) {
	return p.sampleVoices.Start(patch, synth, pitch, velocity)
}

func (p *StepperPlayer) SamplePlaybackStopIfCurrent(event *StepperSamplePlaybackStop) *NoteOff {
	if event == nil {
		return nil
	}
	voice := p.sampleVoices.StopIfCurrent(event.Voice)
	if voice == nil {
		return nil
	}
	return voice.NoteOff()
}

func (p *StepperPlayer) StopSamplesplitterVoice(patch string) {
	voice := p.sampleVoices.StopPatch(patch)
	if voice == nil {
		return
	}
	if noteOff := voice.NoteOff(); noteOff != nil {
		voice.Synth.SendNoteToMidiOutput(noteOff)
	}
}

func (p *StepperPlayer) StopAllSamplesplitterVoices() {
	for _, voice := range p.sampleVoices.StopAll() {
		if noteOff := voice.NoteOff(); noteOff != nil {
			voice.Synth.SendNoteToMidiOutput(noteOff)
		}
	}
}

func (p *StepperPlayer) ResetPitchBends() {
	synths := map[*Synth]bool{}
	for _, patch := range []string{"A", "B", "C", "D"} {
		palettePatch := GetPatch(patch)
		if palettePatch != nil {
			if synth := palettePatch.Synth(); synth != nil {
				synths[synth] = true
			}
		}
		if synth := p.config.SamplesplitterSynthForPatch(patch); synth != nil {
			synths[synth] = true
		}
	}
	for synth := range synths {
		synth.SendPitchBend(MidiPitchBendCenter)
	}
}
