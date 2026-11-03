package kit

import (
	"sync/atomic"
	"testing"
	"time"
)

func previewTestOutput(t *testing.T, state *previewGestureState) (func() int, chan CursorEvent) {
	t.Helper()
	var gid atomic.Int64
	events := make(chan CursorEvent, 256)
	t.Cleanup(func() { state.setEnabled(false) })
	return func() int { return int(gid.Add(1)) }, events
}

func waitPreviewEvent(t *testing.T, events <-chan CursorEvent, ddu string) CursorEvent {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		select {
		case ce := <-events:
			if ce.Ddu == ddu {
				return ce
			}
		case <-deadline.C:
			t.Fatalf("no preview %s event", ddu)
		}
	}
}

func assertPreviewStopped(t *testing.T, state *previewGestureState, events <-chan CursorEvent) {
	t.Helper()
	state.mutex.Lock()
	running := state.run != nil
	state.mutex.Unlock()
	if running {
		t.Fatal("preview gesture is still running")
	}
	select {
	case ce := <-events:
		t.Fatalf("event after preview stopped: %+v", ce)
	case <-time.After(80 * time.Millisecond):
	}
}

func TestPreviewGestureStartsOnlyAfterPresetLoadInPreviewMode(t *testing.T) {
	var state previewGestureState
	nextGID, events := previewTestOutput(t, &state)
	emit := func(ce CursorEvent) { events <- ce }
	state.start(state.beginLoad(), "A", nextGID, emit)
	state.mutex.Lock()
	running := state.run != nil
	state.mutex.Unlock()
	if running {
		t.Fatal("preset load started a gesture outside Preview mode")
	}
	state.setEnabled(true)
	select {
	case <-events:
		t.Fatal("entering Preview must wait for a preset selection")
	default:
	}
	state.start(state.beginLoad(), "A", nextGID, emit)
	down := waitPreviewEvent(t, events, "down")
	drag := waitPreviewEvent(t, events, "drag")
	if drag.Pos == down.Pos || drag.Tag != "A,preview" || drag.GID != down.GID {
		t.Fatalf("expected a moving cursor on A, got %+v then %+v", down, drag)
	}
	if drag.Pos.X < 0 || drag.Pos.X > 1 || drag.Pos.Y < 0 || drag.Pos.Y > 1 || drag.Pos.Z <= 0 || drag.Pos.Z > 1 {
		t.Fatalf("preview position out of range: %+v", drag.Pos)
	}
	state.setEnabled(false)
	up := waitPreviewEvent(t, events, "up")
	if up.GID != down.GID || up.Pos.Z != 0 {
		t.Fatalf("Preview exit did not release its cursor: %+v", up)
	}
	assertPreviewStopped(t, &state, events)
}

func TestPreviewGestureRealInputStopsOnlyTheMatchingPad(t *testing.T) {
	oldPatches := Patchs
	patch := &Patch{name: "A"}
	Patchs = map[string]*Patch{"A": patch, "B": {name: "B"}}
	t.Cleanup(func() { Patchs = oldPatches })
	nextGID, events := previewTestOutput(t, &patch.preview)
	emit := func(ce CursorEvent) { events <- ce }
	patch.preview.setEnabled(true)
	patch.preview.start(patch.preview.beginLoad(), "A", nextGID, emit)
	waitPreviewEvent(t, events, "down")
	stopPreviewGestureForInput(NewCursorEvent(100, "B", "down", CursorPos{}))
	waitPreviewEvent(t, events, "drag")
	stopPreviewGestureForInput(NewCursorEvent(101, "A,morph", "drag", CursorPos{}))
	waitPreviewEvent(t, events, "up")
	assertPreviewStopped(t, &patch.preview, events)
	// A new preset selection explicitly starts another demo after real input.
	patch.preview.start(patch.preview.beginLoad(), "A", nextGID, emit)
	waitPreviewEvent(t, events, "down")
}

func TestPreviewGestureDoesNotOverrideInputDuringPresetLoad(t *testing.T) {
	var state previewGestureState
	nextGID, events := previewTestOutput(t, &state)
	emit := func(ce CursorEvent) { events <- ce }
	state.setEnabled(true)
	ticket := state.beginLoad()
	state.realInput()
	state.start(ticket, "A", nextGID, emit)
	assertPreviewStopped(t, &state, events)
	// Leaving Preview while a load is pending also prevents its later start.
	ticket = state.beginLoad()
	state.setEnabled(false)
	state.start(ticket, "A", nextGID, emit)
	assertPreviewStopped(t, &state, events)
}

func TestPreviewGestureNewPresetReleasesThePreviousCursor(t *testing.T) {
	var state previewGestureState
	nextGID, events := previewTestOutput(t, &state)
	emit := func(ce CursorEvent) { events <- ce }
	state.setEnabled(true)
	state.start(state.beginLoad(), "A", nextGID, emit)
	first := waitPreviewEvent(t, events, "down")
	ticket := state.beginLoad()
	up := waitPreviewEvent(t, events, "up")
	if up.GID != first.GID {
		t.Fatal("new preset did not release the previous preview cursor")
	}
	state.start(ticket, "A", nextGID, emit)
	second := waitPreviewEvent(t, events, "down")
	if second.GID == first.GID {
		t.Fatal("new preview reused the old cursor")
	}
	for i := 0; i < 3; i++ {
		if ce := waitPreviewEvent(t, events, "drag"); ce.GID != second.GID {
			t.Fatalf("old cursor emitted after a new preset: %+v", ce)
		}
	}
}
