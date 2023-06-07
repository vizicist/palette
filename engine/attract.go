package engine

import (
	"math/rand"
	"sync/atomic"
	"time"
)

type AttractManager struct {
	// attractMutex           sync.RWMutex
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
	}
	am.attractModeIsOn.Store(false) // probably not needed?
	return am
}

func (am *AttractManager) AttractModeIsOn() bool {
	// am.attractMutex.Lock()
	isOn := am.attractModeIsOn.Load()
	// am.attractMutex.Unlock()
	return isOn
}

func (am *AttractManager) SetAttractMode(onoff bool) {
	if onoff == am.AttractModeIsOn() {
		LogOfType("attract", "setAttractMode already in mode", "onoff", onoff)
		return // already in that mode
	}
	am.setAttractMode(onoff)
}

func (am *AttractManager) setAttractMode(onoff bool) {

	//// am.attractMutex.Lock()
	//// defer am.attractMutex.Unlock()

	// Throttle it a bit
	secondsSince := time.Since(am.lastAttractModeChange).Seconds()
	if secondsSince > 1.0 {
		// LogOfType("attract", "AttractManager changing attract", "onoff", onoff)
		// am.attractMutex.Lock()
		am.attractModeIsOn.Store(onoff)
		// am.attractMutex.Unlock()
		am.lastAttractModeChange = time.Now()
		LogInfo("setAttractMode", "onoff", onoff)
	} else {
		LogInfo("NOT setting setAttractMode, too quick!", "onoff", onoff)
	}
}

func (am *AttractManager) checkAttract() {

	//// am.attractMutex.Lock()

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()
	// attractModeEnabled := am.attractIdleSecs > 0
	sinceLastAttractCheck := now.Sub(am.lastAttractCheck).Seconds()
	if am.attractEnabled && sinceLastAttractCheck > am.attractCheckSecs {
		am.lastAttractCheck = now
		// There's a delay when checking cursor activity to turn attract mod on.
		// Non-internal cursor activity turns attract mode off instantly.
		sinceLastAttractModeChange := time.Since(am.lastAttractModeChange).Seconds()
		ison := am.AttractModeIsOn()
		if !ison && sinceLastAttractModeChange > am.attractIdleSecs {
			// Nothing happening for a while, turn attract mode on
			//// am.attractMutex.Unlock()
			am.setAttractMode(true)
			//// am.attractMutex.Lock()
		}
	}
	///// am.attractMutex.Unlock()

	if am.AttractModeIsOn() {
		am.doAttractAction()
	}
}

func (am *AttractManager) doAttractAction() {

	//// am.attractMutex.Lock()

	if !am.AttractModeIsOn() {
		//// am.attractMutex.Unlock()
		return
	}
	now := time.Now()
	dt := now.Sub(am.lastAttractGestureTime).Seconds()
	dp := now.Sub(am.lastAttractChange).Seconds()
	if dt > am.attractGestureInterval {
		source := string("ABCD"[rand.Int()%4])
		dur := 2 * time.Second
		go TheCursorManager.GenerateRandomGesture(source, "internal", dur)
		am.lastAttractGestureTime = now
	}

	if dp > am.attractChangeInterval {
		if TheQuadPro == nil {
			LogWarn("No QuadPro to change for attract mode")
		} else {
			err := TheQuadPro.loadQuadRand()
			LogIfError(err)
		}
		am.lastAttractChange = now
	}

	//// am.attractMutex.Unlock()
}
