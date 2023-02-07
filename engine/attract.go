package engine

import (
	"fmt"
	"math/rand"
	"time"
)

type AttractManager struct {
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

	nextCursorNum int
}

var TheAttractManager *AttractManager

func NewAttractManager() *AttractManager {

	am := &AttractManager{
		attractModeIsOn:        false,
		lastAttractModeChange:  time.Now(),
		lastAttractGestureTime: time.Now(),
		lastAttractChange:      time.Now(),
		lastAttractCheck:       time.Now(),
	}
	am.attractCheckSecs = TheEngine.EngineParamFloatWithDefault("engine.attractchecksecs", 2)
	am.attractIdleSecs = TheEngine.EngineParamFloatWithDefault("engine.attractidlesecs", 0)
	am.attractChangeInterval = TheEngine.EngineParamFloatWithDefault("engine.attractchangeinterval", 30)
	am.attractGestureInterval = TheEngine.EngineParamFloatWithDefault("engine.attractgestureinterval", 0.5)

	return am
}

func (am *AttractManager) setAttractMode(onoff bool) {

	if onoff == am.attractModeIsOn {
		return // already in that mode
	}
	// Throttle it a bit
	secondsSince := time.Since(am.lastAttractModeChange).Seconds()
	if secondsSince > 1.0 {
		LogOfType("attract", "AttractManager changing attract", "onoff", onoff)
		am.attractModeIsOn = onoff
		am.lastAttractModeChange = time.Now()
	}
}

func (am *AttractManager) checkAttract() {

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()
	attractModeEnabled := am.attractIdleSecs > 0
	sinceLastAttractCheck := now.Sub(am.lastAttractCheck).Seconds()
	if attractModeEnabled && sinceLastAttractCheck > am.attractCheckSecs {
		am.lastAttractCheck = now
		// There's a delay when checking cursor activity to turn attract mod on.
		// Non-internal cursor activity turns attract mode off instantly.
		sinceLastAttractModeChange := time.Since(am.lastAttractModeChange).Seconds()
		if !am.attractModeIsOn && sinceLastAttractModeChange > am.attractIdleSecs {
			// Nothing happening for a while, turn attract mode on
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

		cid := fmt.Sprintf("%s#%d,internal", source, am.nextCursorNum)
		am.nextCursorNum++

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		noteDuration := time.Second
		go GenerateCursorGesture(cid, noteDuration, x0, y0, z0, x1, y1, z1)
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
