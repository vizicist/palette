package engine

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Scheduler struct {
	Items    []SchedItem
	nextItem int
	// mutex    sync.RWMutex

	activePhrasesManager *ActivePhrasesManager

	now       time.Time
	time0     time.Time
	lastClick Clicks
	control   chan Command

	transposeAuto   bool
	transposeNext   Clicks
	transposeClicks Clicks // time between auto transpose changes
	transposeIndex  int    // current place in tranposeValues
	transposeValues []int

	attractModeIsOn        bool
	lastAttractGestureTime time.Time
	lastAttractPresetTime  time.Time
	attractGestureDuration time.Duration
	attractNoteDuration    time.Duration
	attractPresetDuration  time.Duration
	attractPreset          string
	attractClient          *osc.Client
	lastAttractChange      float64
	attractCheckSecs       float64
	attractIdleSecs        float64
	processCheckSecs       float64
	aliveSecs              float64
}

var currentMilli int64
var currentMilliMutex sync.Mutex
var currentMilliOffset int64
var currentClickOffset Clicks
var clicksPerSecond int
var currentClick Clicks
var oneBeat Clicks
var currentClickMutex sync.Mutex

// CurrentMilli is the time from the start, in milliseconds
const defaultClicksPerSecond = 192
const minClicksPerSecond = (defaultClicksPerSecond / 16)
const maxClicksPerSecond = (defaultClicksPerSecond * 16)

var defaultSynth = "default"

// var loopForever = 999999

// Bits for Events
const EventMidiInput = 0x01
const EventNoteOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventNoteOutput | EventCursor

func CurrentClick() Clicks {
	currentClickMutex.Lock()
	defer currentClickMutex.Unlock()
	return currentClick
}

func SetCurrentClick(clk Clicks) {
	currentClickMutex.Lock()
	currentClick = clk
	currentClickMutex.Unlock()
}

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec int) {
	clicksPerSecond = clkpersec
	currentMilliOffset = 0
	currentClickOffset = 0
	oneBeat = Clicks(clicksPerSecond / 2) // i.e. 120bpm
}

// ChangeClicksPerSecond is what you use to change the tempo
func ChangeClicksPerSecond(factor float64) {
	TempoFactor = factor
	clkpersec := int(defaultClicksPerSecond * factor)
	if clkpersec < minClicksPerSecond {
		clkpersec = minClicksPerSecond
	}
	if clkpersec > maxClicksPerSecond {
		clkpersec = maxClicksPerSecond
	}
	currentMilliOffset = CurrentMilli()
	currentClickOffset = CurrentClick()
	clicksPerSecond = clkpersec
	oneBeat = Clicks(clicksPerSecond / 2)
}

// Seconds2Clicks converts a Time value (elapsed seconds) to Clicks
func Seconds2Clicks(tm float64) Clicks {
	return currentClickOffset + Clicks(0.5+float64(tm*1000-float64(currentMilliOffset))*(float64(clicksPerSecond)/1000.0))
}

// Clicks2Seconds converts Clicks to Time (seconds), relative
func Clicks2Seconds(clk Clicks) float64 {
	return float64(clk) / float64(clicksPerSecond)
}

// Clicks2Seconds converts Clicks to Time (seconds), absolute
func Clicks2SecondsAbsolute(clk Clicks) float64 {
	// Take current*Offset values into account
	clk -= currentClickOffset
	secs := float64(clk) / float64(clicksPerSecond)
	secs -= (float64(currentMilliOffset) * 1000.0)
	return secs
}

// TempoFactor xxx
var TempoFactor = float64(1.0)

func CurrentMilli() int64 {
	currentMilliMutex.Lock()
	defer currentMilliMutex.Unlock()
	return int64(currentMilli)
}

func SetCurrentMilli(m int64) {
	currentMilliMutex.Lock()
	currentMilli = m
	currentMilliMutex.Unlock()
}

type PhraseIter struct {
}

type SchedItem struct {
	ClickStart Clicks
	PhraseIter *PhraseIter
}

func (sched *Scheduler) InsertAtClick(clk Clicks) {

}

func NewScheduler() *Scheduler {
	transposebeats := Clicks(ConfigIntWithDefault("transposebeats", 48))
	s := &Scheduler{
		Items:           []SchedItem{},
		nextItem:        -1,
		transposeAuto:   ConfigBoolWithDefault("transposeauto", true),
		transposeClicks: transposebeats,
		transposeNext:   transposebeats * oneBeat, // first one
		// NOTE: the order in transposeValues needs to match the
		// order in palette.py
		transposeValues:      []int{0, -2, 3, -5},
		activePhrasesManager: NewActivePhrasesManager(),
	}
	return s
}

// Start runs the scheduler and never returns
func (sched *Scheduler) Start() {

	log.Println("Scheduler begins")

	// Wake up every 2 milliseconds and check looper events
	tick := time.NewTicker(2 * time.Millisecond)
	sched.time0 = <-tick.C

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	sched.processCheckSecs = float64(ConfigFloatWithDefault("processchecksecs", 60))
	sched.attractCheckSecs = float64(ConfigFloatWithDefault("attractchecksecs", 2))
	sched.aliveSecs = float64(ConfigFloatWithDefault("alivesecs", 5))
	sched.attractIdleSecs = float64(ConfigFloatWithDefault("attractidlesecs", 0))

	secs1 := ConfigFloatWithDefault("attractpresetduration", 30)
	sched.attractPresetDuration = time.Duration(int(secs1 * float32(time.Second)))

	secs := ConfigFloatWithDefault("attractgestureduration", 0.5)
	sched.attractGestureDuration = time.Duration(int(secs * float32(time.Second)))

	secs = ConfigFloatWithDefault("attractnoteduration", 0.2)
	sched.attractNoteDuration = time.Duration(int(secs * float32(time.Second)))

	sched.attractPreset = ConfigStringWithDefault("attractpreset", "random")

	var lastAttractCheck float64
	var lastProcessCheck float64
	var lastAlive float64

	// By reading from tick.C, we wake up every 2 milliseconds
	for now := range tick.C {
		// log.Printf("Realtime loop now=%v\n", time.Now())
		sched.now = now
		uptimesecs := sched.Uptime()
		newclick := Seconds2Clicks(uptimesecs)
		SetCurrentMilli(int64(uptimesecs * 1000.0))

		currentClick := CurrentClick()
		if newclick > currentClick {
			// log.Printf("ADVANCING CLICK now=%v click=%d\n", time.Now(), newclick)
			sched.advanceTransposeTo(newclick)
			sched.advanceClickTo(currentClick)
			SetCurrentClick(newclick)
		}

		// Every so often we check to see if attract mode should be turned on
		attractModeEnabled := sched.attractIdleSecs > 0
		sinceLastAttractChange := uptimesecs - sched.lastAttractChange
		sinceLastAttractCheck := uptimesecs - lastAttractCheck
		if attractModeEnabled && sinceLastAttractCheck > sched.attractCheckSecs {
			lastAttractCheck = uptimesecs
			// There's a delay when checking cursor activity to turn attract mod on.
			// Non-internal cursor activity turns attract mode off instantly.
			if !sched.attractModeIsOn && sinceLastAttractChange > sched.attractIdleSecs {
				// Nothing happening for a while, turn attract mode on
				sched.AttractMode(true)
			}
		}

		sinceLastAlive := uptimesecs - lastAlive
		if sinceLastAlive > sched.aliveSecs {
			sched.publishOscAlive(uptimesecs)
			lastAlive = uptimesecs
		}

		if sched.attractModeIsOn {
			sched.doAttractAction()
		}

		// At the beginning (lastProcessCheck==0)
		// and then every so often (ie. processCheckSecs)
		// we check to see if necessary processes are still running
		sinceLastProcessCheck := uptimesecs - lastProcessCheck
		processCheckEnabled := sched.processCheckSecs > 0
		if processCheckEnabled && (lastProcessCheck == 0 || sinceLastProcessCheck > sched.processCheckSecs) {
			// Put it in background, so calling
			// tasklist or ps doesn't disrupt realtime
			if Debug.Realtime {
				log.Printf("StartRealtime: checking processes\n")
			}
			go TheEngine.ProcessManager.checkProcessesAndRestartIfNecessary()
			lastProcessCheck = uptimesecs
		}

		select {
		case cmd := <-sched.control:
			_ = cmd
			log.Println("Realtime got command on control channel: ", cmd)
		default:
		}
	}
	log.Println("StartRealtime ends")
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// Don't let events get handled while we're advancing
	TheEngine.Router.eventMutex.Lock()
	defer TheEngine.Router.eventMutex.Unlock()

	for clk := sched.lastClick; clk < toClick; clk++ {
		sched.activePhrasesManager.AdvanceByOneClick()
		TheEngine.CursorManager.CheckCursorUp(time.Now())
	}
	sched.lastClick = toClick
}

/*
func (sched *Scheduler) advanceByOneClick() {
}
*/

// checkDelay is the Duration that has to pass
// before we decide a cursor is no longer present,
// resulting in a cursor UP event.
var checkDelay time.Duration = 0

/*
// Time returns the current time
func (sched *Scheduler) time() time.Time {
	return time.Now()
}
*/

/*
func (sched *Scheduler) checkCursorUp(cm *CursorManager) {
	now := sched.now
	cm.CheckCursorUp(now)
}
*/

/*
func (motor *Motor) handleCursorDeviceEvent(e CursorDeviceEvent) {

	id := e.Source

	router := TheEngine.Router
	router.deviceCursorsMutex.Lock()
	defer router.deviceCursorsMutex.Unlock()

	// e.Ddu is "down", "drag", or "up"

	tc, ok := motor.deviceCursors[id]
	if !ok {
		// new DeviceCursor
		tc = &DeviceCursor{}
		motor.deviceCursors[id] = tc
	}
	tc.lastTouch = motor.time()

	// If it's a new (downed==false) cursor, make sure the first step event is "down
	if !tc.downed {
		e.Ddu = "down"
		tc.downed = true
	}
	cse := CursorStepEvent{
		ID:  id,
		X:   e.X,
		Y:   e.Y,
		Z:   e.Z,
		Ddu: e.Ddu,
	}
	if Debug.Cursor {
		log.Printf("Motor.handleCursorDeviceEvent: pad=%s id=%s ddu=%s xyz=%.4f,%.4f,%.4f\n", motor.padName, id, e.Ddu, e.X, e.Y, e.Z)
	}

	motor.executeIncomingCursor(cse)

	if e.Ddu == "up" {
		// if Debug.Cursor {
		// 	log.Printf("Router.handleCursorDeviceEvent: deleting cursor id=%s\n", id)
		// }
		delete(motor.deviceCursors, id)
	}
}
*/

func (sched *Scheduler) AttractMode(on bool) {
	if sched.attractModeIsOn == on {
		// no change
		return
	}
	sched.attractModeIsOn = on
	log.Printf("AttractMode: changing to %v\n", on)
	sched.lastAttractChange = sched.Uptime()
}

func (sched *Scheduler) Uptime() float64 {
	return sched.now.Sub(sched.time0).Seconds()
}

func (sched *Scheduler) publishOscAlive(uptimesecs float64) {
	attractMode := sched.attractModeIsOn
	if Debug.Attract {
		log.Printf("publishOscAlive: uptime=%v attract=%v\n", uptimesecs, attractMode)
	}
	if sched.attractClient == nil {
		sched.attractClient = osc.NewClient("127.0.0.1", 3331)
	}
	msg := osc.NewMessage("/alive")
	msg.Append(float32(uptimesecs))
	msg.Append(attractMode)
	err := sched.attractClient.Send(msg)
	if err != nil {
		log.Printf("publishOscAlive: Send err=%s\n", err)
	}
}

func (sched *Scheduler) doAttractAction() {

	now := time.Now()
	dt := now.Sub(sched.lastAttractGestureTime)
	if sched.attractModeIsOn && dt > sched.attractGestureDuration {
		log.Printf("doAttractAction: doing stuff\n")
		regions := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		region := regions[i]
		sched.lastAttractGestureTime = now

		cid := fmt.Sprintf("%d", time.Now().UnixNano())

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		go TheEngine.CursorManager.doCursorGesture(region, cid, x0, y0, z0, x1, y1, z1)
		sched.lastAttractGestureTime = now
	}

	dp := now.Sub(sched.lastAttractPresetTime)
	if sched.attractPreset == "random" && dp > TheEngine.Scheduler.attractPresetDuration {
		TheEngine.Router.loadQuadPresetRand()
		sched.lastAttractPresetTime = now
	}
}

func (sched *Scheduler) advanceTransposeTo(newclick Clicks) {
	if sched.transposeAuto && sched.transposeNext < newclick {
		sched.transposeNext += (sched.transposeClicks * oneBeat)
		sched.transposeIndex = (sched.transposeIndex + 1) % len(sched.transposeValues)
		transposePitch := sched.transposeValues[sched.transposeIndex]
		if Debug.Transpose {
			log.Printf("advanceTransposeTo: newclick=%d transposePitch=%d\n", newclick, transposePitch)
		}
		/*
			for _, motor := range TheEngine.Router.motors {
				// motor.clearDown()
				if Debug.Transpose {
					log.Printf("  setting transposepitch in motor pad=%s trans=%d activeNotes=%d\n", motor.padName, transposePitch, len(motor.activeNotes))
				}
				motor.TransposePitch = transposePitch
			}
		*/
		sched.activePhrasesManager.terminateActiveNotes()
	}
}
