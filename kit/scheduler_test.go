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

// A panic on the realtime path must cost one tick, not the scheduler.
//
// The recover used to sit at Start's scope, outside the `for range tick.C`
// loop, so any panic unwound the loop and Start returned. Nothing restarts that
// goroutine, so the engine kept answering HTTP with its clock stopped: no
// scheduled notes, no cursor handling, no attract, and no hung-note watchdog,
// which lives in the same loop. Nothing outside could tell, because the process
// was still up.
func TestSchedulerTickSurvivesPanic(t *testing.T) {
	InitLog("")

	oldScheduler := theScheduler
	oldAttract := theAttractManager
	oldClick := CurrentClick()
	defer func() {
		theScheduler = oldScheduler
		theAttractManager = oldAttract
		SetCurrentClick(oldClick)
	}()

	sched := NewScheduler()
	theScheduler = sched

	// An AttractManager whose atomic flag was never allocated: checkAttract
	// dereferences it. This stands in for the reachable panics on this path -
	// a nil Synth in the MIDI-thru lookup, or loadQuadRand taking a modulo of
	// an empty quad directory.
	theAttractManager = &AttractManager{}

	// Make sure the tick gets past its "clock hasn't advanced" early return.
	SetCurrentClick(-1000)

	// If this propagates, the test binary dies right here - which is precisely
	// what happened to the scheduler goroutine before the fix.
	sched.tickOnce()

	if sched.tickPanics != 1 {
		t.Fatalf("tickOnce recovered %d panics, want 1 (did the tick actually panic?)", sched.tickPanics)
	}

	// And the loop keeps going: the next tick still runs.
	sched.tickOnce()
	if sched.tickPanics != 2 {
		t.Fatalf("after a second panicking tick, recovered %d, want 2", sched.tickPanics)
	}
}
