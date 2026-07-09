package kit

import "testing"

type schedulingMidiOut struct {
	fakeMidiOut
	onSend func()
}

func (out *schedulingMidiOut) Send(data []byte) error {
	out.sent = append(out.sent, append([]byte(nil), data...))
	if out.onSend != nil {
		out.onSend()
	}
	return nil
}

func TestSchedulerDrainsNewlyScheduledDueEventsInSameClick(t *testing.T) {
	synth, _, cleanupSynth := setupSynthWatchdogTest(t)
	defer cleanupSynth()

	oldScheduler := theScheduler
	oldClick := CurrentClick()
	defer func() {
		theScheduler = oldScheduler
		SetCurrentClick(oldClick)
	}()

	sched := NewScheduler()
	theScheduler = sched
	SetCurrentClick(100)

	output := &schedulingMidiOut{fakeMidiOut: fakeMidiOut{open: true}}
	synth.state.output = output
	output.onSend = func() {
		if len(output.sent) == 1 {
			ScheduleAt(CurrentClick(), "A", NewNoteOn(synth, 61, 100))
		}
	}

	ScheduleAt(CurrentClick(), "A", NewNoteOn(synth, 60, 100))
	sched.triggerClickAndDrain(CurrentClick())

	if got := len(output.sent); got != 2 {
		t.Fatalf("sent MIDI messages = %d, want 2", got)
	}
	if got := output.sent[0][1]; got != 60 {
		t.Fatalf("first MIDI pitch = %d, want 60", got)
	}
	if got := output.sent[1][1]; got != 61 {
		t.Fatalf("second MIDI pitch = %d, want 61", got)
	}
	if got := sched.schedList.Len(); got != 0 {
		t.Fatalf("scheduled events remaining = %d, want 0", got)
	}
	if got := pendingCount(sched); got != 0 {
		t.Fatalf("pending events remaining = %d, want 0", got)
	}
}

func TestSchedulerDoesNotDrainFutureEventsEarly(t *testing.T) {
	synth, _, cleanupSynth := setupSynthWatchdogTest(t)
	defer cleanupSynth()

	oldScheduler := theScheduler
	oldClick := CurrentClick()
	defer func() {
		theScheduler = oldScheduler
		SetCurrentClick(oldClick)
	}()

	sched := NewScheduler()
	theScheduler = sched
	SetCurrentClick(200)

	output := &schedulingMidiOut{fakeMidiOut: fakeMidiOut{open: true}}
	synth.state.output = output
	output.onSend = func() {
		if len(output.sent) == 1 {
			ScheduleAt(CurrentClick()+1, "A", NewNoteOn(synth, 61, 100))
		}
	}

	ScheduleAt(CurrentClick(), "A", NewNoteOn(synth, 60, 100))
	sched.triggerClickAndDrain(CurrentClick())

	if got := len(output.sent); got != 1 {
		t.Fatalf("MIDI messages at click 200 = %d, want 1", got)
	}
	if got := sched.schedList.Len(); got != 1 {
		t.Fatalf("future scheduled events = %d, want 1", got)
	}

	SetCurrentClick(201)
	sched.triggerClickAndDrain(CurrentClick())
	if got := len(output.sent); got != 2 {
		t.Fatalf("MIDI messages through click 201 = %d, want 2", got)
	}
}

func pendingCount(sched *Scheduler) int {
	sched.pendingMutex.RLock()
	defer sched.pendingMutex.RUnlock()
	return len(sched.pendingScheduled)
}
