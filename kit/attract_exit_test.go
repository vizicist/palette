package kit

import (
	"sync/atomic"
	"testing"
	"time"
)

// attractManagerOn returns a manager that is enabled and currently in attract
// mode, with the mode having changed just now - the state you are in when the
// attract screen has only just appeared.
func attractManagerOn() *AttractManager {
	InitLog("") // setAttractMode logs, which needs a logger
	am := &AttractManager{
		attractEnabled:        true,
		attractModeIsOn:       &atomic.Bool{},
		lastAttractModeChange: time.Now(),
		ModeCheckSecs:         2,
	}
	am.attractModeIsOn.Store(true)
	return am
}

// Playing the pads must drop attract mode straight away, even though the mode
// only just turned on. The throttle used to swallow this.
func TestAttractModeTurnsOffImmediately(t *testing.T) {
	old := theAttractManager
	defer func() { theAttractManager = old }()
	theAttractManager = attractManagerOn()

	theAttractManager.SetAttractMode(false)

	if theAttractManager.AttractModeIsOn() {
		t.Fatal("attract mode still on after being turned off")
	}
}

// A finger on the pads produces a stream of events, and each one resets the
// idle timer. That used to keep the throttle window permanently closed, so
// attract mode never turned off while someone was playing.
func TestAttractModeTurnsOffDuringContinuousInput(t *testing.T) {
	old := theAttractManager
	defer func() { theAttractManager = old }()
	am := attractManagerOn()
	theAttractManager = am

	// Mimic quad.onCursorEvent: try to leave attract mode, then reset the
	// idle timer, over and over as a drag would.
	for i := 0; i < 10; i++ {
		if am.AttractModeIsOn() {
			am.SetAttractMode(false)
		}
		am.attractMutex.Lock()
		am.lastAttractModeChange = time.Now()
		am.attractMutex.Unlock()

		if !am.AttractModeIsOn() {
			return // left attract mode, which is the point
		}
	}
	t.Fatal("attract mode survived a stream of cursor events")
}

// Turning it on stays throttled, so it can't flap back on right after being
// dismissed.
func TestAttractModeTurningOnIsStillThrottled(t *testing.T) {
	old := theAttractManager
	defer func() { theAttractManager = old }()
	InitLog("")
	am := &AttractManager{
		attractEnabled:        true,
		attractModeIsOn:       &atomic.Bool{},
		lastAttractModeChange: time.Now(), // changed just now
	}
	theAttractManager = am

	am.SetAttractMode(true)

	if am.AttractModeIsOn() {
		t.Fatal("attract mode turned on inside the throttle window")
	}
}

func TestAttractModeTurnsOnOnceThrottleHasElapsed(t *testing.T) {
	old := theAttractManager
	defer func() { theAttractManager = old }()
	InitLog("")
	am := &AttractManager{
		attractEnabled:        true,
		attractModeIsOn:       &atomic.Bool{},
		lastAttractModeChange: time.Now().Add(-5 * time.Second),
	}
	theAttractManager = am

	am.SetAttractMode(true)

	if !am.AttractModeIsOn() {
		t.Fatal("attract mode did not turn on after the throttle window")
	}
}

// Repeated "turn it off" calls are harmless: the first one changes the mode
// and the rest are no-ops, so nothing flaps.
func TestAttractModeOffIsIdempotent(t *testing.T) {
	old := theAttractManager
	defer func() { theAttractManager = old }()
	am := attractManagerOn()
	theAttractManager = am

	for i := 0; i < 5; i++ {
		am.SetAttractMode(false)
	}
	if am.AttractModeIsOn() {
		t.Fatal("attract mode on after repeated off calls")
	}
}
