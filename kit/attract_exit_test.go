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

// attractManagerWithTouchThreshold returns a manager in attract mode that needs
// `needed` touches within `secs` seconds before it will let go.
func attractManagerWithTouchThreshold(needed int, secs float64) *AttractManager {
	am := attractManagerOn()
	am.ExitTouchCount = needed
	am.ExitTouchSecs = secs
	return am
}

// One touch must not end attract mode. Someone brushing past the pads, or a
// stray reading from the depth camera, would otherwise drop the installation
// out of its attract loop.
func TestAttractTouchThresholdNeedsSeveralTouches(t *testing.T) {
	am := attractManagerWithTouchThreshold(3, 2.0)

	if am.noticeTouch() {
		t.Fatal("one touch ended attract mode")
	}
	if am.noticeTouch() {
		t.Fatal("two touches ended attract mode")
	}
	if !am.noticeTouch() {
		t.Fatal("three touches within the window did not end attract mode")
	}
}

// Touches that have aged out of the window don't count, so a slow drip of stray
// input never accumulates into an exit.
func TestAttractTouchThresholdForgetsOldTouches(t *testing.T) {
	am := attractManagerWithTouchThreshold(3, 2.0)
	am.attractTouchTimes = []time.Time{
		time.Now().Add(-10 * time.Second),
		time.Now().Add(-5 * time.Second),
	}

	if am.noticeTouch() {
		t.Fatal("touches from outside the window counted towards leaving attract mode")
	}
	if got := len(am.attractTouchTimes); got != 1 {
		t.Fatalf("kept %d touches, want only the one just now", got)
	}
}

// Reaching the threshold clears the history, so the touch after it starts a
// fresh count instead of ending attract mode all over again.
func TestAttractTouchThresholdResetsAfterReaching(t *testing.T) {
	am := attractManagerWithTouchThreshold(2, 2.0)

	am.noticeTouch()
	if !am.noticeTouch() {
		t.Fatal("two touches did not reach the threshold")
	}
	if am.noticeTouch() {
		t.Fatal("a single touch after the threshold ended attract mode again")
	}
}

// A mode change drops the history, so touches from before it can't count
// towards leaving attract mode the next time it turns on.
func TestAttractForgetTouches(t *testing.T) {
	am := attractManagerWithTouchThreshold(3, 2.0)

	am.noticeTouch()
	am.noticeTouch()
	am.forgetTouches()

	if am.noticeTouch() {
		t.Fatal("touches from before the reset still counted")
	}
}

// An unset or nonsense count behaves the way it did before any of this: the
// first touch is enough.
func TestAttractTouchThresholdUnsetCountExitsOnFirstTouch(t *testing.T) {
	am := attractManagerWithTouchThreshold(0, 2.0)

	if !am.noticeTouch() {
		t.Fatal("with no threshold configured, one touch should end attract mode")
	}
}
