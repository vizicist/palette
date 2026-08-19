package kit

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// attractSettings is the tunable half of the attract configuration.
//
// It is copied out of the manager under attractMutex so a reader gets one
// consistent set of values without holding the lock across the work it drives.
// These used to be plain exported fields that the API goroutine wrote - every
// global.attract* parameter has a setter - while the scheduler and the cursor
// paths read them with no synchronisation at all.
type attractSettings struct {
	Enabled              bool
	ModeCheckSecs        float64
	GestureMinLength     float64
	GestureMaxLength     float64
	GestureZMin          float64
	GestureZMax          float64
	GestureNumSteps      int
	GestureDuration      float64
	GestureInterval      float64
	PresetChangeInterval float64
	IdleSecs             float64
	ExitTouchCount       int
	ExitTouchSecs        float64
}

type AttractManager struct {
	// attractMutex guards settings and every lastAttract* time below. The
	// times matter most: time.Time is three words, so an unsynchronised read
	// of one being written can return an instant that never existed.
	attractMutex            sync.RWMutex
	settings                attractSettings
	attractModeIsOn         *atomic.Bool
	lastAttractModeChange   time.Time
	lastAttractGestureTime  time.Time
	lastAttractPresetChange time.Time
	lastAttractModeCheck    time.Time

	attractRand      *rand.Rand
	attractRandMutex sync.Mutex

	// Recent touches on the pads, used to decide whether someone is really
	// there. Only touches inside ExitTouchSecs are kept.
	attractTouchMutex sync.Mutex
	attractTouches    []attractTouch
}

// Settings returns a copy of the tunables.
func (am *AttractManager) Settings() attractSettings {
	am.attractMutex.RLock()
	defer am.attractMutex.RUnlock()
	return am.settings
}

// updateSettings applies a change to the tunables under the lock.
func (am *AttractManager) updateSettings(fn func(s *attractSettings)) {
	am.attractMutex.Lock()
	defer am.attractMutex.Unlock()
	fn(&am.settings)
}

// noteAttractActivity resets the idle timer. Called for every real cursor
// event, from the cursor goroutine.
func (am *AttractManager) noteAttractActivity() {
	am.attractMutex.Lock()
	am.lastAttractModeChange = time.Now()
	am.attractMutex.Unlock()
}

var theAttractManager *AttractManager

func NewAttractManager() *AttractManager {

	am := &AttractManager{
		attractMutex:            sync.RWMutex{},
		attractModeIsOn:         &atomic.Bool{},
		lastAttractModeChange:   time.Now(),
		lastAttractGestureTime:  time.Now(),
		lastAttractPresetChange: time.Now(),
		lastAttractModeCheck:    time.Now(),
		attractRand:             rand.New(rand.NewSource(time.Now().UnixNano())),
		attractRandMutex:        sync.Mutex{},
		settings:                attractSettings{ModeCheckSecs: 2},
	}

	// paramFloat/paramInt log (with the param name) and return 0 on error.
	paramFloat := func(name string) float64 {
		v, err := GetParamFloat(name)
		LogIfError(err, "param", name)
		return v
	}
	paramInt := func(name string) int {
		v, err := GetParamInt(name)
		LogIfError(err, "param", name)
		return v
	}

	am.updateSettings(func(s *attractSettings) {
		s.GestureInterval = paramFloat("global.attractgestureinterval")
		s.GestureMinLength = paramFloat("global.attractgestureminlength")
		s.GestureMaxLength = paramFloat("global.attractgesturemaxlength")
		s.GestureZMin = paramFloat("global.attractgesturezmin")
		s.GestureZMax = paramFloat("global.attractgesturezmax")
		s.GestureNumSteps = paramInt("global.attractgesturenumsteps")
		s.GestureDuration = paramFloat("global.attractgestureduration")
		s.PresetChangeInterval = paramFloat("global.attractpresetchangeinterval")
		s.IdleSecs = paramFloat("global.attractidlesecs")
		s.ExitTouchCount = paramInt("global.attractexittouches")
		s.ExitTouchSecs = paramFloat("global.attractexitsecs")
	})

	return am
}

// attractTouch is one contact on the pads: the cursor GID that produced it, and
// when that contact was first seen.
type attractTouch struct {
	gid  int
	when time.Time
}

// attractExitSecsDefault stands in for global.attractexitsecs when that
// parameter can't be read, matching the paramdef's init.
const attractExitSecsDefault = 3.0

// noticeTouch records a touch on the pads from one cursor GID, and reports
// whether enough distinct contacts have arrived close enough together to mean
// someone is really there.
//
// A single touch used to be enough, which made attract mode fragile: someone
// brushing past the pads, or one stray reading from the depth camera, would
// drop the installation out of its attract loop and leave it sitting idle until
// the idle timer brought it back.
//
// Counting by GID rather than by event is what makes that work. A Morph
// re-sends "down" for a contact that is still held, and ExecuteCursorEvent
// carries that through unchanged, so one press arrives as a stream of down
// events rather than one: a capture of a single press showed the same GID
// firing six of them inside a second. Counting those raw reaches any threshold
// in a fraction of a second, which is no better than the single-touch
// behaviour this replaces. One contact is one touch, however many events it
// emits, so the count means what it says.
func (am *AttractManager) noticeTouch(gid int) bool {

	cfg := am.Settings()
	needed := cfg.ExitTouchCount
	if needed < 1 {
		needed = 1 // an unset or nonsense parameter behaves as it did before
	}

	// The window needs the same defence as the count, and more urgently: at zero
	// every touch ages out before the next one arrives, so the list never grows
	// past one and any needed above 1 can never be reached - the pads would stop
	// being able to end attract mode at all. The paramdef's minimum is 0.5, so
	// this only comes up when the parameter can't be read, which is exactly when
	// nothing else will catch it.
	within := cfg.ExitTouchSecs
	if within <= 0 {
		within = attractExitSecsDefault
	}

	am.attractTouchMutex.Lock()
	defer am.attractTouchMutex.Unlock()

	// Drop the touches that have aged out, so the count always means "this many
	// within the last ExitTouchSecs" rather than a running total that a slow
	// drip of stray input would eventually reach.
	now := time.Now()
	alreadyCounted := false
	kept := am.attractTouches[:0]
	for _, touch := range am.attractTouches {
		if now.Sub(touch.when).Seconds() > within {
			continue
		}
		if touch.gid == gid {
			alreadyCounted = true
		}
		kept = append(kept, touch)
	}
	am.attractTouches = kept

	if !alreadyCounted {
		am.attractTouches = append(am.attractTouches, attractTouch{gid: gid, when: now})
	}

	if len(am.attractTouches) < needed {
		return false
	}
	am.attractTouches = am.attractTouches[:0]
	return true
}

// forgetTouches drops the touch history, so touches from before a mode change
// can't count towards leaving attract mode the next time it turns on.
func (am *AttractManager) forgetTouches() {
	am.attractTouchMutex.Lock()
	am.attractTouches = am.attractTouches[:0]
	am.attractTouchMutex.Unlock()
}

func (am *AttractManager) SetAttractEnabled(b bool) {
	am.updateSettings(func(s *attractSettings) { s.Enabled = b })
	// Disabling attract mode is also an instruction to leave it. Keeping the
	// raw mode bit set would make a later "off" request look like a no-op and
	// could leave the attract video layer soloed indefinitely.
	if !b && am.attractModeIsOn.Load() {
		am.setAttractMode(false)
	}
}

func (am *AttractManager) AttractModeIsOn() bool {
	// Enabled controls automatic idle entry and generated gestures, not the
	// actual mode state. Manual entry (the Show Goats button) must work while
	// automatic attract mode is disabled, and callers must still be able to
	// observe and turn off that manually-entered mode.
	return am.attractModeIsOn.Load()
}

func (am *AttractManager) SetAttractMode(onoff bool) {
	if onoff == am.AttractModeIsOn() {
		LogWarn("setAttractMode already in mode", "onoff", onoff)
		return // already in that mode
	}
	// By the time this is called with false, someone has definitely asked for
	// it: either the API, or enough touches on the pads to satisfy noticeTouch.
	// So it happens immediately. Throttling it would leave the attract screen up
	// while the pads are being played, because every cursor event also resets
	// lastAttractModeChange as the idle timer: while a finger keeps moving, the
	// throttle window never elapses.
	//
	// Turning it on is automatic and idle-driven, so that stays throttled to
	// keep it from flapping.
	if !onoff {
		am.setAttractMode(false)
		return
	}
	am.attractMutex.RLock()
	secondsSince := time.Since(am.lastAttractModeChange).Seconds()
	am.attractMutex.RUnlock()
	if secondsSince > 1.0 {
		am.setAttractMode(onoff)
	} else {
		LogWarn("NOT setting setAttractMode, too quick!", "onoff", onoff)
	}
}

// resetBidule restarts Bidule's transport. It is indirected through a variable
// so tests can check which attract transitions reach it - the whole point of
// the change that introduced this is which ones do not. The default spawns the
// goroutine itself, both because Reset sleeps while it waits for Bidule and
// setAttractMode runs on the scheduler tick, and so that a test replacing this
// observes a plain synchronous call.
var resetBidule = func() { go TheBidule().Reset() }

func (am *AttractManager) setAttractMode(onoff bool) {

	LogInfo("setAttractMode", "onoff", onoff)

	am.attractModeIsOn.Store(onoff)
	am.forgetTouches()

	if theQuad != nil {
		for _, patch := range Patchs {
			patch.clearGraphics()
			patch.loopClear()
		}
		// Silence in both directions, across every route rather than just the
		// MIDI synths. Leaving attract matters because it drives the pads with
		// generated gestures and can leave a voice sounding; entering it
		// matters because a note left hanging as someone walks away would
		// otherwise play on underneath the attract screen.
		theQuad.allNotesOff()
	}

	// Bidule gets reset on the way in only. Reset switches Bidule's transport
	// off, sleeps 400ms, and switches it back on - and attract mode is left
	// precisely because someone is touching the pads, so on the way out that
	// toggle is guaranteed to run underneath the first notes of a new session.
	// A note-on landing while the transport is cycling can leave a voice
	// sounding in Bidule whose note-off it never acts on. Nothing catches that:
	// the engine sent both the on and the off, so its own bookkeeping is
	// balanced and the 30-second watchdog never sees it (SendExpiredNoteOffs
	// only expires notes it still has recorded as down). The note then drones
	// until the next attract entry happens to send an unconditional ANO.
	//
	// Entering attract is the safe moment for the same work, since nobody is
	// playing - that is why the mode turned on. Nothing is lost by skipping it
	// on the way out: Reset finishes by turning the transport back on, so it is
	// already on by the time attract ends, and allNotesOff above still runs in
	// both directions, so leaving attract is still silent.
	if onoff {
		resetBidule()
	}

	// Attract videos, on installations that have them, play on the Resolume
	// output for as long as attract mode lasts. Both calls reach Resolume over
	// the network, so they run off the tick like the Bidule reset.
	if onoff {
		go TheAttractVideoPlayer().Start()
	} else {
		go TheAttractVideoPlayer().Stop()
	}

	am.attractMutex.Lock()
	am.lastAttractModeChange = time.Now()
	am.attractMutex.Unlock()

	// Tell the GUI, so the attract screen appears and disappears with the
	// mode. Without this only the API path notified, so attract mode turned
	// off by playing the pads left the attract screen showing.
	NotifyStatusChanged()
}

func (am *AttractManager) checkAttract() {

	// Attract videos advance whenever attract mode is on, however it got there.
	// This sits above the global.attractenabled gate on purpose: that parameter
	// governs attract mode arriving on its own after an idle timeout, and the
	// generated gestures once it has, but the Show Goats button turns attract
	// mode on by hand and global.attractenabled ships off. Behind the gate, the
	// button would put the first video up and then leave it there for good.
	if am.AttractModeIsOn() {
		TheAttractVideoPlayer().Advance()
	}

	cfg := am.Settings()
	if !cfg.Enabled {
		return
	}

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()

	am.attractMutex.Lock()
	sinceLastAttractModeCheck := now.Sub(am.lastAttractModeCheck).Seconds()
	dueForCheck := sinceLastAttractModeCheck > cfg.ModeCheckSecs
	idleTooLong := false
	if dueForCheck {
		am.lastAttractModeCheck = now
		idleTooLong = time.Since(am.lastAttractModeChange).Seconds() > cfg.IdleSecs
	}
	am.attractMutex.Unlock()

	if dueForCheck {
		ison := am.AttractModeIsOn()
		if !ison && idleTooLong {
			am.setAttractMode(true)
		}
	}

	if am.AttractModeIsOn() {
		am.doAttractAction()
	}
}

func (am *AttractManager) doAttractAction() {

	cfg := am.Settings()
	if !cfg.Enabled || !am.AttractModeIsOn() {
		return
	}

	now := time.Now()

	am.attractMutex.Lock()
	dt := now.Sub(am.lastAttractGestureTime).Seconds()
	dueForGesture := dt > cfg.GestureInterval
	if dueForGesture {
		am.lastAttractGestureTime = now
	}
	dp := now.Sub(am.lastAttractPresetChange).Seconds()
	dueForPreset := dp > cfg.PresetChangeInterval
	if dueForPreset {
		am.lastAttractPresetChange = now
	}
	am.attractMutex.Unlock()

	if dueForGesture {

		// Start a random gesture
		am.attractRandMutex.Lock()
		patch := string("ABCD"[am.attractRand.Intn(len(Patchs))])
		am.attractRandMutex.Unlock()

		tag := patch + ",attract"
		dur := time.Duration(cfg.GestureDuration * float64(time.Second))

		go theCursorManager.GenerateRandomGesture(tag, cfg.GestureNumSteps, dur)
	}

	if dueForPreset {
		if theQuad == nil {
			LogWarn("No Quad to change for attract mode")
		} else {
			_, err := theQuad.loadQuadRand("quad")
			LogIfError(err)
		}
	}
}
