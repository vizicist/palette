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
	mutex    sync.RWMutex

	time      time.Time
	time0     time.Time
	lastClick Clicks
	control   chan Command

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
var loopForever = 999999

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

type SchedItem struct {
	ClickStart Clicks
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		Items:    []SchedItem{},
		nextItem: -1,
	}
	return s
}

// StartRealtime runs the looper and never returns
func (r *Scheduler) Start() {

	log.Println("StartRealtime begins")

	// Wake up every 2 milliseconds and check looper events
	tick := time.NewTicker(2 * time.Millisecond)
	r.time0 = <-tick.C

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	r.processCheckSecs = float64(ConfigFloatWithDefault("processchecksecs", 60))
	r.attractCheckSecs = float64(ConfigFloatWithDefault("attractchecksecs", 2))
	r.aliveSecs = float64(ConfigFloatWithDefault("alivesecs", 5))
	r.attractIdleSecs = float64(ConfigFloatWithDefault("attractidlesecs", 0))

	secs1 := ConfigFloatWithDefault("attractpresetduration", 30)
	r.attractPresetDuration = time.Duration(int(secs1 * float32(time.Second)))

	secs := ConfigFloatWithDefault("attractgestureduration", 0.5)
	r.attractGestureDuration = time.Duration(int(secs * float32(time.Second)))

	secs = ConfigFloatWithDefault("attractnoteduration", 0.2)
	r.attractNoteDuration = time.Duration(int(secs * float32(time.Second)))

	r.attractPreset = ConfigStringWithDefault("attractpreset", "random")

	var lastAttractCheck float64
	var lastProcessCheck float64
	var lastAlive float64

	// By reading from tick.C, we wake up every 2 milliseconds
	for now := range tick.C {
		// log.Printf("Realtime loop now=%v\n", time.Now())
		r.time = now
		uptimesecs := r.Uptime()
		newclick := Seconds2Clicks(uptimesecs)
		SetCurrentMilli(int64(uptimesecs * 1000.0))

		currentClick := CurrentClick()
		if newclick > currentClick {
			// log.Printf("ADVANCING CLICK now=%v click=%d\n", time.Now(), newclick)
			r.advanceTransposeTo(newclick)
			r.advanceClickTo(currentClick)
			SetCurrentClick(newclick)
		}

		// Every so often we check to see if attract mode should be turned on
		attractModeEnabled := r.attractIdleSecs > 0
		sinceLastAttractChange := uptimesecs - r.lastAttractChange
		sinceLastAttractCheck := uptimesecs - lastAttractCheck
		if attractModeEnabled && sinceLastAttractCheck > r.attractCheckSecs {
			lastAttractCheck = uptimesecs
			// There's a delay when checking cursor activity to turn attract mod on.
			// Non-internal cursor activity turns attract mode off instantly.
			if !r.attractModeIsOn && sinceLastAttractChange > r.attractIdleSecs {
				// Nothing happening for a while, turn attract mode on
				r.AttractMode(true)
			}
		}

		sinceLastAlive := uptimesecs - lastAlive
		if sinceLastAlive > r.aliveSecs {
			r.publishOscAlive(uptimesecs)
			lastAlive = uptimesecs
		}

		if r.attractModeIsOn {
			r.doAttractAction()
		}

		// At the beginning (lastProcessCheck==0)
		// and then every so often (ie. processCheckSecs)
		// we check to see if necessary processes are still running
		sinceLastProcessCheck := uptimesecs - lastProcessCheck
		processCheckEnabled := r.processCheckSecs > 0
		if processCheckEnabled && (lastProcessCheck == 0 || sinceLastProcessCheck > r.processCheckSecs) {
			// Put it in background, so calling
			// tasklist or ps doesn't disrupt realtime
			if Debug.Realtime {
				log.Printf("StartRealtime: checking processes\n")
			}
			go r.CheckProcessesAndRestartIfNecessary()
			lastProcessCheck = uptimesecs
		}

		select {
		case cmd := <-r.control:
			_ = cmd
			log.Println("Realtime got command on control channel: ", cmd)
		default:
		}
	}
	log.Println("StartRealtime ends")
}

func (r *Scheduler) AttractMode(on bool) {
	if r.attractModeIsOn == on {
		// no change
		return
	}
	r.attractModeIsOn = on
	log.Printf("AttractMode: changing to %v\n", on)
	r.lastAttractChange = r.Uptime()
}

func (r *Scheduler) Uptime() float64 {
	return r.time.Sub(r.time0).Seconds()
}

func (r *Scheduler) publishOscAlive(uptimesecs float64) {
	attractMode := r.attractModeIsOn
	if Debug.Attract {
		log.Printf("publishOscAlive: uptime=%v attract=%v\n", uptimesecs, attractMode)
	}
	if r.attractClient == nil {
		r.attractClient = osc.NewClient("127.0.0.1", 3331)
	}
	msg := osc.NewMessage("/alive")
	msg.Append(float32(uptimesecs))
	msg.Append(attractMode)
	err := r.attractClient.Send(msg)
	if err != nil {
		log.Printf("publishOscAlive: Send err=%s\n", err)
	}
}

func (r *Router) doAttractAction() {

	now := time.Now()
	dt := now.Sub(r.lastAttractGestureTime)
	if r.attractModeIsOn && dt > r.attractGestureDuration {
		log.Printf("doAttractAction: doing stuff\n")
		regions := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		region := regions[i]
		r.lastAttractGestureTime = now

		cid := fmt.Sprintf("%d", time.Now().UnixNano())

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		go r.doCursorGesture(region, cid, x0, y0, z0, x1, y1, z1)
		r.lastAttractGestureTime = now
	}

	dp := now.Sub(r.lastAttractPresetTime)
	if r.attractPreset == "random" && dp > r.attractPresetDuration {
		r.loadQuadPresetRand()
		r.lastAttractPresetTime = now
	}
}
