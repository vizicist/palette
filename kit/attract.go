package kit

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type AttractManager struct {
	attractMutex           sync.RWMutex
	attractEnabled         bool
	attractModeIsOn        *atomic.Bool
	lastAttractModeChange  time.Time
	lastAttractGestureTime time.Time
	lastAttractChange      time.Time

	// parameters
	attractGestureInterval float64
	attractChangeInterval  float64
	lastAttractCheck       time.Time
	attractCheckSecs       float64
	attractIdleSecs        float64

	attractRand      *rand.Rand
	attractRandMutex sync.Mutex
}

var TheAttractManager *AttractManager

func NewAttractManager() *AttractManager {

	am := &AttractManager{
		attractEnabled:         false,
		attractModeIsOn:        &atomic.Bool{},
		lastAttractModeChange:  time.Now(),
		lastAttractGestureTime: time.Now(),
		lastAttractChange:      time.Now(),
		lastAttractCheck:       time.Now(),
		attractCheckSecs:       2,
		attractIdleSecs:        70,
		attractChangeInterval:  30,
		attractGestureInterval: 0.5,
		attractRand:            rand.New(rand.NewSource(1)),
	}

	am.attractModeIsOn.Store(false) // probably not needed?

	return am
}

func (am *AttractManager) SetAttractEnabled(b bool) {
	am.attractEnabled = b
}

func (am *AttractManager) AttractModeIsOn() bool {
	// am.attractMutex.Lock()
	isOn := am.attractModeIsOn.Load()
	// am.attractMutex.Unlock()
	return isOn && am.attractEnabled
}

func (am *AttractManager) SetAttractMode(onoff bool) {
	if onoff == am.AttractModeIsOn() {
		LogWarn("setAttractMode already in mode", "onoff", onoff)
		return // already in that mode
	}
	// Throttle it a bit
	secondsSince := time.Since(am.lastAttractModeChange).Seconds()
	if secondsSince > 1.0 {
		am.setAttractMode(onoff)
	} else {
		LogWarn("NOT setting setAttractMode, too quick!", "onoff", onoff)
	}
}

func (am *AttractManager) setAttractMode(onoff bool) {

	LogInfo("setAttractMode", "onoff", onoff)

	//// am.attractMutex.Lock()
	// am.attractMutex.Lock()
	am.attractModeIsOn.Store(onoff)

	if TheQuad != nil {
		for _, patch := range Patchs {
			patch.clearGraphics()
			patch.loopClear()
		}
	}

	go TheBidule().Reset()

	// am.attractMutex.Unlock()
	am.lastAttractModeChange = time.Now()
}

func (am *AttractManager) checkAttract() {

	if !am.attractEnabled {
		return
	}
	//// am.attractMutex.Lock()

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()
	// attractModeEnabled := am.attractIdleSecs > 0
	sinceLastAttractCheck := now.Sub(am.lastAttractCheck).Seconds()
	if sinceLastAttractCheck > am.attractCheckSecs {

		am.lastAttractCheck = now

		am.attractMutex.Lock()
		sinceLastAttractModeChange := time.Since(am.lastAttractModeChange).Seconds()
		ison := am.AttractModeIsOn()
		idleTooLong := sinceLastAttractModeChange > am.attractIdleSecs
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

	//// am.attractMutex.Lock()

	if !am.attractEnabled || !am.AttractModeIsOn() {
		//// am.attractMutex.Unlock()
		return
	}
	now := time.Now()
	dt := now.Sub(am.lastAttractGestureTime).Seconds()

	am.attractRandMutex.Lock()
	patch := string("ABCD"[am.attractRand.Intn(len(Patchs))])
	am.attractRandMutex.Unlock()

	numsteps, err := GetParamInt("global.attractgesturenumsteps")
	if err != nil {
		LogIfError(err)
		return
	}
	durfloat, err := GetParamFloat("global.attractgestureduration")
	if err != nil {
		LogIfError(err)
		return
	}
	dur := time.Duration(durfloat * float64(time.Second))

	tag := patch + ",attract"
	if dt > am.attractGestureInterval {
		go TheCursorManager.GenerateRandomGesture(tag, numsteps, dur)
		am.lastAttractGestureTime = now
	}

	dp := now.Sub(am.lastAttractChange).Seconds()
	if dp > am.attractChangeInterval {
		if TheQuad == nil {
			LogWarn("No Quad to change for attract mode")
		} else {
			_, err := TheQuad.loadQuadRand("quad")
			LogIfError(err)
		}
		am.lastAttractChange = now
	}

	//// am.attractMutex.Unlock()
}
