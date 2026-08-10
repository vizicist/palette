package kit

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type AttractManager struct {
	attractMutex            sync.RWMutex
	attractEnabled          bool
	attractModeIsOn         *atomic.Bool
	lastAttractModeChange   time.Time
	lastAttractGestureTime  time.Time
	lastAttractPresetChange time.Time
	lastAttractModeCheck    time.Time
	ModeCheckSecs           float64

	attractRand      *rand.Rand
	attractRandMutex sync.Mutex

	// parameters
	GestureMinLength     float64
	GestureMaxLength     float64
	GestureZMin          float64
	GestureZMax          float64
	GestureNumSteps      int
	GestureDuration      float64
	GestureInterval      float64
	PresetChangeInterval float64
	IdleSecs             float64
}

var theAttractManager *AttractManager

func NewAttractManager() *AttractManager {

	am := &AttractManager{
		attractMutex:            sync.RWMutex{},
		attractEnabled:          false,
		attractModeIsOn:         &atomic.Bool{},
		lastAttractModeChange:   time.Now(),
		lastAttractGestureTime:  time.Now(),
		lastAttractPresetChange: time.Now(),
		lastAttractModeCheck:    time.Now(),
		ModeCheckSecs:           2,
		attractRand:             rand.New(rand.NewSource(time.Now().UnixNano())),
		attractRandMutex:        sync.Mutex{},

		GestureMinLength:     0,
		GestureMaxLength:     0,
		GestureZMin:          0,
		GestureZMax:          0,
		GestureNumSteps:      0,
		GestureDuration:      0,
		GestureInterval:      0,
		PresetChangeInterval: 0,
		IdleSecs:             0,
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

	am.GestureInterval = paramFloat("global.attractgestureinterval")
	am.GestureMinLength = paramFloat("global.attractgestureminlength")
	am.GestureMaxLength = paramFloat("global.attractgesturemaxlength")
	am.GestureZMin = paramFloat("global.attractgesturezmin")
	am.GestureZMax = paramFloat("global.attractgesturezmax")
	am.GestureNumSteps = paramInt("global.attractgesturenumsteps")
	am.GestureDuration = paramFloat("global.attractgestureduration")
	am.PresetChangeInterval = paramFloat("global.attractpresetchangeinterval")
	am.IdleSecs = paramFloat("global.attractidlesecs")

	return am
}

func (am *AttractManager) SetAttractEnabled(b bool) {
	am.attractEnabled = b
}

func (am *AttractManager) AttractModeIsOn() bool {
	isOn := am.attractModeIsOn.Load()
	return isOn && am.attractEnabled
}

func (am *AttractManager) SetAttractMode(onoff bool) {
	if onoff == am.AttractModeIsOn() {
		LogWarn("setAttractMode already in mode", "onoff", onoff)
		return // already in that mode
	}
	// Turning attract mode off is always someone asking for it - a hand on the
	// pads, or the API - so it happens immediately. Throttling it would leave
	// the attract screen up while the pads are being played, because every
	// cursor event also resets lastAttractModeChange as the idle timer: while
	// a finger keeps moving, the throttle window never elapses.
	//
	// Turning it on is automatic and idle-driven, so that stays throttled to
	// keep it from flapping.
	if !onoff {
		am.setAttractMode(false)
		return
	}
	secondsSince := time.Since(am.lastAttractModeChange).Seconds()
	if secondsSince > 1.0 {
		am.setAttractMode(onoff)
	} else {
		LogWarn("NOT setting setAttractMode, too quick!", "onoff", onoff)
	}
}

func (am *AttractManager) setAttractMode(onoff bool) {

	LogInfo("setAttractMode", "onoff", onoff)

	am.attractModeIsOn.Store(onoff)

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

	go TheBidule().Reset()

	// Attract videos, on installations that have them, play on the Resolume
	// output for as long as attract mode lasts. Both calls reach Resolume over
	// the network, so they run off the tick like the Bidule reset above.
	if onoff {
		go TheAttractVideoPlayer().Start()
	} else {
		go TheAttractVideoPlayer().Stop()
	}

	am.lastAttractModeChange = time.Now()

	// Tell the GUI, so the attract screen appears and disappears with the
	// mode. Without this only the API path notified, so attract mode turned
	// off by playing the pads left the attract screen showing.
	NotifyStatusChanged()
}

func (am *AttractManager) checkAttract() {

	if !am.attractEnabled {
		return
	}

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()
	sinceLastAttractModeCheck := now.Sub(am.lastAttractModeCheck).Seconds()
	if sinceLastAttractModeCheck > am.ModeCheckSecs {

		am.lastAttractModeCheck = now

		am.attractMutex.Lock()
		sinceLastAttractModeChange := time.Since(am.lastAttractModeChange).Seconds()
		ison := am.AttractModeIsOn()
		idleTooLong := sinceLastAttractModeChange > am.IdleSecs
		am.attractMutex.Unlock()

		if !ison && idleTooLong {
			am.setAttractMode(true)
		}
	}

	if am.AttractModeIsOn() {
		am.doAttractAction()
	}
}

func (am *AttractManager) doAttractAction() {

	if !am.attractEnabled || !am.AttractModeIsOn() {
		return
	}
	// Move to the next video when the current one has played through. This only
	// sends OSC, so it is cheap enough to check on every attract tick.
	TheAttractVideoPlayer().Advance()

	now := time.Now()
	dt := now.Sub(am.lastAttractGestureTime).Seconds()

	if dt > am.GestureInterval {

		// Start a random gesture
		am.attractRandMutex.Lock()
		patch := string("ABCD"[am.attractRand.Intn(len(Patchs))])
		am.attractRandMutex.Unlock()

		tag := patch + ",attract"
		am.lastAttractGestureTime = now

		dur := time.Duration(am.GestureDuration * float64(time.Second))

		go theCursorManager.GenerateRandomGesture(tag, am.GestureNumSteps, dur)
	}

	dp := now.Sub(am.lastAttractPresetChange).Seconds()
	if dp > am.PresetChangeInterval {
		if theQuad == nil {
			LogWarn("No Quad to change for attract mode")
		} else {
			_, err := theQuad.loadQuadRand("quad")
			LogIfError(err)
		}
		am.lastAttractPresetChange = now
	}

}
