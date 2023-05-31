package engine

import (
	"fmt"
	"math/rand"
	"time"
)

type AttractManager struct {
	attractEnabled         bool
	attractModeIsOn        bool
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
		attractModeIsOn:        false,
		lastAttractModeChange:  time.Now(),
		lastAttractGestureTime: time.Now(),
		lastAttractChange:      time.Now(),
		lastAttractCheck:       time.Now(),
		attractCheckSecs:       2,
		attractIdleSecs:        70,
		attractChangeInterval:  30,
		attractGestureInterval: 0.5,
	}
	return am
}

func (am *AttractManager) CurrentAttractMode() bool{
	return am.attractModeIsOn
}

func (am *AttractManager) SetAttractMode(onoff bool) {
	if onoff == am.attractModeIsOn {
		LogOfType("attract","setAttractMode already in mode", "attractModeIsOn", am.attractModeIsOn)
		return // already in that mode
	}
	am.setAttractMode(onoff)
}

func (am *AttractManager) setAttractMode(onoff bool) {

	// Throttle it a bit
	secondsSince := time.Since(am.lastAttractModeChange).Seconds()
	if secondsSince > 1.0 {
		// LogOfType("attract", "AttractManager changing attract", "onoff", onoff)
		am.attractModeIsOn = onoff
		am.lastAttractModeChange = time.Now()
		LogInfo("setAttractMode", "attractModeIsOn", am.attractModeIsOn)
	} else {
		LogInfo("NOT setting setAttractMode, too quick!", "onoff", onoff)
	}
}

func (am *AttractManager) checkAttract() {

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()
	// attractModeEnabled := am.attractIdleSecs > 0
	sinceLastAttractCheck := now.Sub(am.lastAttractCheck).Seconds()
	if am.attractEnabled && sinceLastAttractCheck > am.attractCheckSecs {
		am.lastAttractCheck = now
		// There's a delay when checking cursor activity to turn attract mod on.
		// Non-internal cursor activity turns attract mode off instantly.
		sinceLastAttractModeChange := time.Since(am.lastAttractModeChange).Seconds()
		if !am.attractModeIsOn && sinceLastAttractModeChange > am.attractIdleSecs {
			// Nothing happening for a while, turn attract mode on
			LogInfo("CHECKATTRACT setting attractmode to true")
			am.setAttractMode(true)
		}
	}

	if am.attractModeIsOn {
		am.doAttractAction()
	}
}

func (am *AttractManager) doAttractAction() {

	now := time.Now()
	dt := now.Sub(am.lastAttractGestureTime).Seconds()
	if am.attractModeIsOn && dt > am.attractGestureInterval {

		sourceNames := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		source := sourceNames[i]
		am.lastAttractGestureTime = now

		cid := fmt.Sprintf("%s#%d,internal", source, TheCursorManager.UniqueInt())

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		noteDuration := time.Second
		go TheCursorManager.GenerateCursorGesture(cid, noteDuration, x0, y0, z0, x1, y1, z1)
		am.lastAttractGestureTime = now
	}

	dp := now.Sub(am.lastAttractChange).Seconds()
	if dp > am.attractChangeInterval {
		if TheQuadPro == nil {
			LogWarn("No QuadPro to change for attract mode")
		} else {
			err := TheQuadPro.loadQuadRand()
			LogIfError(err)
		}
		am.lastAttractChange = now
	}
}
