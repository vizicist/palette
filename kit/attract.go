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
	GestureNumSteps      float64
	GestureDuration      float64
	GestureInterval      float64
	PresetChangeInterval float64
	IdleSecs             float64
}

var TheAttractManager *AttractManager

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

	var err error

	am.GestureInterval, err = GetParamFloat("global.attractgestureinterval")
	LogIfError(err)
	am.GestureInterval, err = GetParamFloat("global.attractgestureinterval")
	LogIfError(err)
	am.GestureMinLength, err = GetParamFloat("global.attractgestureminlength")
	LogIfError(err)
	am.GestureMaxLength, err = GetParamFloat("global.attractgesturemaxlength")
	LogIfError(err)
	am.GestureZMin, err = GetParamFloat("global.attractgesturezmin")
	LogIfError(err)
	am.GestureZMax, err = GetParamFloat("global.attractgesturezmax")
	LogIfError(err)
	am.GestureNumSteps, err = GetParamFloat("global.attractgesturenumsteps")
	LogIfError(err)
	am.GestureDuration, err = GetParamFloat("global.attractgestureduration")
	LogIfError(err)
	am.GestureInterval, err = GetParamFloat("global.attractgestureinterval")
	LogIfError(err)
	am.PresetChangeInterval, err = GetParamFloat("global.attractpresetchangeinterval")	
	LogIfError(err)
	am.IdleSecs, err = GetParamFloat("global.attractidlesecs")
	LogIfError(err)

	// am.attractModeIsOn.Store(false) // probably not needed?

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

	NatsPublishFromEngine("attract", map[string]any{
		"onoff": onoff,
	})

	if TheQuad != nil {
		for _, patch := range Patchs {
			patch.clearGraphics()
			patch.loopClear()
		}
		if !onoff {
			for _, patch := range TheQuad.patch {
				patch.Synth().SendANO()
			}
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

	//// am.attractMutex.Lock()

	if !am.attractEnabled || !am.AttractModeIsOn() {
		//// am.attractMutex.Unlock()
		return
	}
	now := time.Now()
	dt := now.Sub(am.lastAttractGestureTime).Seconds()

	if dt > am.GestureInterval {

		// Start a random gesture
		am.attractRandMutex.Lock()
		patch := string("ABCD"[am.attractRand.Intn(len(Patchs))])
		am.attractRandMutex.Unlock()

		tag := patch + ",attract"
		am.lastAttractGestureTime = now

		numsteps := int(am.GestureNumSteps)
		dur := time.Duration(am.GestureDuration * float64(time.Second))

		go TheCursorManager.GenerateRandomGesture(tag, numsteps, dur)
	}

	dp := now.Sub(am.lastAttractPresetChange).Seconds()
	if dp > am.PresetChangeInterval {
		if TheQuad == nil {
			LogWarn("No Quad to change for attract mode")
		} else {
			_, err := TheQuad.loadQuadRand("quad")
			LogIfError(err)
		}
		am.lastAttractPresetChange = now
	}

	//// am.attractMutex.Unlock()
}
