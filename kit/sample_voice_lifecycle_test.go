package kit

import "testing"

func TestSampleVoiceLifecycleStartReplacesPreviousVoice(t *testing.T) {
	lifecycle := NewSampleVoiceLifecycle()
	synth := &Synth{name: "test"}

	previous, first := lifecycle.Start("A", synth, 60, 100)
	if previous != nil {
		t.Fatalf("first start returned previous voice: %+v", previous)
	}
	if first.Token == 0 {
		t.Fatalf("first voice token was not assigned")
	}

	previous, second := lifecycle.Start("A", synth, 64, 110)
	if previous == nil {
		t.Fatalf("second start did not return replaced voice")
	}
	if previous.Token != first.Token || previous.Pitch != first.Pitch {
		t.Fatalf("previous voice = %+v, want first voice %+v", previous, first)
	}
	if second.Token == first.Token {
		t.Fatalf("replacement voice reused token %d", second.Token)
	}
	if lifecycle.ActiveCount() != 1 {
		t.Fatalf("active count = %d, want 1", lifecycle.ActiveCount())
	}
}

func TestSampleVoiceLifecycleIgnoresStaleStop(t *testing.T) {
	lifecycle := NewSampleVoiceLifecycle()
	synth := &Synth{name: "test"}

	_, first := lifecycle.Start("A", synth, 60, 100)
	_, second := lifecycle.Start("A", synth, 64, 110)

	if stopped := lifecycle.StopIfCurrent(first); stopped != nil {
		t.Fatalf("stale stop stopped voice: %+v", stopped)
	}
	if lifecycle.ActiveCount() != 1 {
		t.Fatalf("active count after stale stop = %d, want 1", lifecycle.ActiveCount())
	}

	stopped := lifecycle.StopIfCurrent(second)
	if stopped == nil {
		t.Fatalf("current stop did not stop voice")
	}
	if stopped.Token != second.Token {
		t.Fatalf("stopped voice token = %d, want %d", stopped.Token, second.Token)
	}
	if lifecycle.ActiveCount() != 0 {
		t.Fatalf("active count after current stop = %d, want 0", lifecycle.ActiveCount())
	}
}

func TestSampleVoiceLifecycleStopsPatchAndAll(t *testing.T) {
	lifecycle := NewSampleVoiceLifecycle()
	synth := &Synth{name: "test"}

	_, voiceA := lifecycle.Start("A", synth, 60, 100)
	_, voiceB := lifecycle.Start("B", synth, 62, 100)

	stoppedA := lifecycle.StopPatch("A")
	if stoppedA == nil || stoppedA.Token != voiceA.Token {
		t.Fatalf("StopPatch stopped %+v, want %+v", stoppedA, voiceA)
	}
	if lifecycle.StopPatch("A") != nil {
		t.Fatalf("StopPatch returned a voice for an already-stopped patch")
	}

	stoppedAll := lifecycle.StopAll()
	if len(stoppedAll) != 1 {
		t.Fatalf("StopAll returned %d voices, want 1", len(stoppedAll))
	}
	if stoppedAll[0].Token != voiceB.Token {
		t.Fatalf("StopAll voice = %+v, want %+v", stoppedAll[0], voiceB)
	}
	if lifecycle.ActiveCount() != 0 {
		t.Fatalf("active count after StopAll = %d, want 0", lifecycle.ActiveCount())
	}
}

func TestStepperSamplePlaybackStopUsesLifecycleTokens(t *testing.T) {
	stepper := NewStepper()
	synth := &Synth{name: "test"}

	_, first := stepper.setActiveSamplesplitterVoice("A", synth, 60, 100)
	_, second := stepper.setActiveSamplesplitterVoice("A", synth, 64, 110)

	if noteOff := stepper.SamplePlaybackStopIfCurrent(&StepperSamplePlaybackStop{Voice: first}); noteOff != nil {
		t.Fatalf("stale scheduled stop produced note off: %+v", noteOff)
	}

	noteOff := stepper.SamplePlaybackStopIfCurrent(&StepperSamplePlaybackStop{Voice: second})
	if noteOff == nil {
		t.Fatalf("current scheduled stop produced no note off")
	}
	if noteOff.Synth != synth || noteOff.Pitch != second.Pitch || noteOff.Velocity != second.Velocity {
		t.Fatalf("note off = %+v, want synth=%p pitch=%d velocity=%d", noteOff, synth, second.Pitch, second.Velocity)
	}
	if stepper.sampleVoices.ActiveCount() != 0 {
		t.Fatalf("active voices after current stop = %d, want 0", stepper.sampleVoices.ActiveCount())
	}
}
