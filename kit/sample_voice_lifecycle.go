package kit

import "sync"

type StepperSampleVoice struct {
	Patch    string
	Synth    *Synth
	Pitch    uint8
	Velocity uint8
	Token    uint64
}

type StepperSamplePlaybackStop struct {
	Voice StepperSampleVoice
}

type SampleVoiceLifecycle struct {
	mutex  sync.Mutex
	voices map[string]StepperSampleVoice
	token  uint64
}

func NewSampleVoiceLifecycle() *SampleVoiceLifecycle {
	return &SampleVoiceLifecycle{
		voices: map[string]StepperSampleVoice{},
	}
}

func (l *SampleVoiceLifecycle) Start(patch string, synth *Synth, pitch uint8, velocity uint8) (*StepperSampleVoice, StepperSampleVoice) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var previous *StepperSampleVoice
	if voice, ok := l.voices[patch]; ok {
		prev := voice
		previous = &prev
	}
	l.token++
	current := StepperSampleVoice{
		Patch:    patch,
		Synth:    synth,
		Pitch:    pitch,
		Velocity: velocity,
		Token:    l.token,
	}
	l.voices[patch] = current
	return previous, current
}

func (l *SampleVoiceLifecycle) StopIfCurrent(voice StepperSampleVoice) *StepperSampleVoice {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	current, ok := l.voices[voice.Patch]
	if !ok || current.Token != voice.Token {
		return nil
	}
	delete(l.voices, voice.Patch)
	stopped := current
	return &stopped
}

func (l *SampleVoiceLifecycle) StopPatch(patch string) *StepperSampleVoice {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	voice, ok := l.voices[patch]
	if !ok {
		return nil
	}
	delete(l.voices, patch)
	stopped := voice
	return &stopped
}

func (l *SampleVoiceLifecycle) StopAll() []StepperSampleVoice {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	voices := make([]StepperSampleVoice, 0, len(l.voices))
	for patch, voice := range l.voices {
		voices = append(voices, voice)
		delete(l.voices, patch)
	}
	return voices
}

func (l *SampleVoiceLifecycle) ActiveCount() int {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return len(l.voices)
}

func (voice StepperSampleVoice) NoteOff() *NoteOff {
	if voice.Synth == nil {
		return nil
	}
	return NewNoteOff(voice.Synth, voice.Pitch, voice.Velocity)
}
