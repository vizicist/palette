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

// global.attractenabled controls automatic idle entry, but Show Goats enters
// attract mode explicitly and must work with that setting off.
func TestManualAttractModeWorksWhileAutomaticModeIsDisabled(t *testing.T) {
	old := theAttractManager
	defer func() { theAttractManager = old }()
	InitLog("")
	am := &AttractManager{
		attractModeIsOn:       &atomic.Bool{},
		lastAttractModeChange: time.Now().Add(-5 * time.Second),
	}
	theAttractManager = am

	am.SetAttractMode(true)

	if !am.AttractModeIsOn() {
		t.Fatal("manual attract mode did not turn on while automatic mode was disabled")
	}
	am.SetAttractMode(false)
}

func TestDisablingAttractModeLeavesActiveMode(t *testing.T) {
	old := theAttractManager
	defer func() { theAttractManager = old }()
	am := attractManagerOn()
	theAttractManager = am

	am.SetAttractEnabled(false)

	if am.attractModeIsOn.Load() {
		t.Fatal("disabling attract mode left the raw mode state on")
	}
	if am.AttractModeIsOn() {
		t.Fatal("attract mode still reported on after it was disabled")
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

	if am.noticeTouch(1) {
		t.Fatal("one touch ended attract mode")
	}
	if am.noticeTouch(2) {
		t.Fatal("two touches ended attract mode")
	}
	if !am.noticeTouch(3) {
		t.Fatal("three touches within the window did not end attract mode")
	}
}

// One contact held on a pad re-sends "down" - a capture of a single press
// showed the same GID firing six of them inside a second. However many events
// it produces, it is one touch.
func TestAttractTouchThresholdCountsOneContactOnce(t *testing.T) {
	am := attractManagerWithTouchThreshold(3, 2.0)

	for i := 0; i < 10; i++ {
		if am.noticeTouch(7) {
			t.Fatalf("a single held contact ended attract mode after %d down events", i+1)
		}
	}
	// Two more distinct contacts are what it takes.
	if am.noticeTouch(15) {
		t.Fatal("two contacts ended attract mode")
	}
	if !am.noticeTouch(115) {
		t.Fatal("three distinct contacts did not end attract mode")
	}
}

// Touches that have aged out of the window don't count, so a slow drip of stray
// input never accumulates into an exit. A contact that comes back after the
// window counts again, since by then it is a new touch.
func TestAttractTouchThresholdForgetsOldTouches(t *testing.T) {
	am := attractManagerWithTouchThreshold(3, 2.0)
	am.attractTouches = []attractTouch{
		{gid: 1, when: time.Now().Add(-10 * time.Second)},
		{gid: 2, when: time.Now().Add(-5 * time.Second)},
	}

	if am.noticeTouch(3) {
		t.Fatal("touches from outside the window counted towards leaving attract mode")
	}
	if got := len(am.attractTouches); got != 1 {
		t.Fatalf("kept %d touches, want only the one just now", got)
	}
}

// Reaching the threshold clears the history, so the touch after it starts a
// fresh count instead of ending attract mode all over again.
func TestAttractTouchThresholdResetsAfterReaching(t *testing.T) {
	am := attractManagerWithTouchThreshold(2, 2.0)

	am.noticeTouch(1)
	if !am.noticeTouch(2) {
		t.Fatal("two touches did not reach the threshold")
	}
	if am.noticeTouch(3) {
		t.Fatal("a single touch after the threshold ended attract mode again")
	}
}

// A mode change drops the history, so touches from before it can't count
// towards leaving attract mode the next time it turns on.
func TestAttractForgetTouches(t *testing.T) {
	am := attractManagerWithTouchThreshold(3, 2.0)

	am.noticeTouch(1)
	am.noticeTouch(2)
	am.forgetTouches()

	if am.noticeTouch(3) {
		t.Fatal("touches from before the reset still counted")
	}
}

// An unset or nonsense count behaves the way it did before any of this: the
// first touch is enough.
func TestAttractTouchThresholdUnsetCountExitsOnFirstTouch(t *testing.T) {
	am := attractManagerWithTouchThreshold(0, 2.0)

	if !am.noticeTouch(1) {
		t.Fatal("with no threshold configured, one touch should end attract mode")
	}
}

// An unset window must not lock the installation into attract mode. At zero
// every touch ages out before the next arrives, so the count never got past one
// and the pads could never end attract mode at all.
func TestAttractTouchThresholdUnsetWindowStillExits(t *testing.T) {
	for _, secs := range []float64{0, -1} {
		am := attractManagerWithTouchThreshold(3, secs)

		if am.noticeTouch(1) {
			t.Fatalf("secs=%v: one touch ended attract mode", secs)
		}
		if am.noticeTouch(2) {
			t.Fatalf("secs=%v: two touches ended attract mode", secs)
		}
		if !am.noticeTouch(3) {
			t.Fatalf("secs=%v: three touches did not end attract mode", secs)
		}
	}
}
