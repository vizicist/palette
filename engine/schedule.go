package engine

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Scheduler struct {
	firstItem *SchedItem
	lastItem  *SchedItem
	// mutex    sync.RWMutex
	nextSid int

	activePhrasesManager *ActivePhrasesManager

	now       time.Time
	time0     time.Time
	lastClick Clicks
	Control   chan Command

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

type SchedItem struct {
	clickStart Clicks
	ID         string
	phrase     *Phrase
	prev       *SchedItem
	next       *SchedItem
}

func (sched *Scheduler) FindInsert(click Clicks) *SchedItem {
	si := sched.firstItem
	for si != nil {
		if si.clickStart >= click {
			return si
		}
		si = si.next
	}
	return nil
}

func (sched *Scheduler) Format(f fmt.State, c rune) {
	s := "Schedule{"
	for si := sched.firstItem; si != nil; si = si.next {
		s += fmt.Sprintf("(%d,%s,%s)", si.clickStart, si.ID, si.phrase)
	}
	s += "}"
	f.Write([]byte(s))
}

func (sched *Scheduler) ScheduleNoteAt(nt *Note, click Clicks) (id string) {
	Info("Scheduler.ScheduleNoteAt", "nt", nt, "click", click)
	phr := NewPhrase().InsertNote(nt)
	id = fmt.Sprintf("%d", sched.nextSid)
	newItem := &SchedItem{
		clickStart: click,
		phrase:     phr,
		prev:       nil,
		next:       nil,
		ID:         id,
	}
	sched.nextSid += 1
	// empty list
	if sched.firstItem == nil {
		sched.firstItem = newItem
		sched.lastItem = newItem
		return
	}
	// Quick check for common situation, adding after lastItem
	if sched.lastItem.clickStart <= click {
		lastlast := sched.lastItem
		sched.lastItem = newItem
		newItem.prev = lastlast
		lastlast.next = newItem
		return
	}
	// otherwise, find the item before which will insert this newItem
	insertBefore := sched.FindInsert(click)
	if insertBefore == nil {
		// The common situation should have already handled this
		Warn("ScheduleNoteAt: Unexpected insert = nil?")
		return
	}
	newItem.prev = insertBefore.prev
	newItem.next = insertBefore
	if insertBefore.prev == nil {
		// it's the new first item
		sched.firstItem = newItem
	}
	insertBefore.prev.next = newItem
	insertBefore.prev = newItem
	return
}

func NewScheduler() *Scheduler {
	transposebeats := Clicks(ConfigIntWithDefault("transposebeats", 48))
	s := &Scheduler{
		firstItem:              nil,
		lastItem:               nil,
		nextSid:                0,
		activePhrasesManager:   NewActivePhrasesManager(),
		now:                    time.Time{},
		time0:                  time.Time{},
		lastClick:              0,
		Control:                make(chan Command),
		transposeAuto:          ConfigBoolWithDefault("transposeauto", true),
		transposeNext:          transposebeats * oneBeat,
		transposeClicks:        transposebeats,
		transposeIndex:         0,
		transposeValues:        []int{0, -2, 3, -5},
		attractModeIsOn:        false,
		lastAttractGestureTime: time.Time{},
		lastAttractPresetTime:  time.Time{},
		attractGestureDuration: 0,
		attractNoteDuration:    0,
		attractPresetDuration:  0,
		attractPreset:          "",
		attractClient:          &osc.Client{},
		lastAttractChange:      0,
		attractCheckSecs:       0,
		attractIdleSecs:        0,
		processCheckSecs:       0,
		aliveSecs:              0,
	}
	return s
}

func SecondsToDuration(secs float32) time.Duration {
	return time.Duration(secs * float32(time.Second.Nanoseconds()))
}

func NanosecsToDuration(nanosecs int64) time.Duration {
	return time.Duration(nanosecs * int64(time.Second.Nanoseconds()))
}

// Start runs the scheduler and never returns
func (sched *Scheduler) Start() {

	Info("Scheduler begins")

	// Wake up every 2 milliseconds and check looper events
	tick := time.NewTicker(2 * time.Millisecond)
	sched.time0 = <-tick.C

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	sched.processCheckSecs = float64(ConfigFloatWithDefault("processchecksecs", 60))
	sched.attractCheckSecs = float64(ConfigFloatWithDefault("attractchecksecs", 2))
	sched.aliveSecs = float64(ConfigFloatWithDefault("alivesecs", 5))
	sched.attractIdleSecs = float64(ConfigFloatWithDefault("attractidlesecs", 0))

	secs1 := int64(ConfigFloatWithDefault("attractpresetduration", 10))
	sched.attractPresetDuration = NanosecsToDuration(secs1)

	secs := ConfigFloatWithDefault("attractgestureduration", 0.5)
	sched.attractGestureDuration = SecondsToDuration(secs)

	secs = ConfigFloatWithDefault("attractnoteduration", 0.2)
	sched.attractNoteDuration = SecondsToDuration(secs)

	sched.attractPreset = ConfigStringWithDefault("attractpreset", "random")

	var lastAttractCheck float64
	var lastProcessCheck float64
	var lastAlive float64

	// By reading from tick.C, we wake up every 2 milliseconds
	for now := range tick.C {
		sched.now = now
		uptimesecs := sched.Uptime()
		newclick := Seconds2Clicks(uptimesecs)
		SetCurrentMilli(int64(uptimesecs * 1000.0))

		currentClick := CurrentClick()
		if newclick > currentClick {
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
				go func() {
					DebugLogOfType("attract", "sending attractmode true to sched.Control")
					sched.Control <- Command{"attractmode", true}
				}()
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
			DebugLogOfType("realtime", "StartRealtime: checking processes")
			go TheEngine().ProcessManager.checkProcessesAndRestartIfNecessary()
			lastProcessCheck = uptimesecs
		}

		select {
		case cmd := <-sched.Control:
			DebugLogOfType("attract", "cmd from sched.Control", "cmd", cmd)
			switch cmd.Action {
			case "attractmode":
				onoff := cmd.Arg.(bool)
				if sched.attractModeIsOn != onoff {
					sched.attractModeIsOn = onoff
					sched.lastAttractChange = sched.Uptime()
				}
			}
		default:
		}
	}
	Info("StartRealtime ends")
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// Don't let events get handled while we're advancing
	TheEngine().Router.eventMutex.Lock()
	defer func() {
		TheEngine().Router.eventMutex.Unlock()
	}()

	for clk := sched.lastClick; clk < toClick; clk++ {
		for si := sched.firstItem; si != nil; si = si.next {
			if si.clickStart == clk {
				sched.activePhrasesManager.StartPhraseAt(si.clickStart, si.phrase, si.ID)
			}
		}
		sched.activePhrasesManager.AdvanceByOneClick()
		TheEngine().CursorManager.checkCursorUp(time.Now())
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
func (player *Player) handleCursorEvent(e CursorEvent) {

	id := e.Source

	router := TheEngine().Router
	router.deviceCursorsMutex.Lock()
	defer router.deviceCursorsMutex.Unlock()

	// e.Ddu is "down", "drag", or "up"

	tc, ok := player.deviceCursors[id]
	if !ok {
		// new DeviceCursor
		tc = &DeviceCursor{}
		player.deviceCursors[id] = tc
	}
	tc.lastTouch = player.time()

	// If it's a new (downed==false) cursor, make sure the first step event is "down"
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

	player.executeIncomingCursor(cse)

	if e.Ddu == "up" {
		delete(player.deviceCursors, id)
	}
}
*/

func (sched *Scheduler) Uptime() float64 {
	return sched.now.Sub(sched.time0).Seconds()
}

func (sched *Scheduler) publishOscAlive(uptimesecs float64) {
	attractMode := sched.attractModeIsOn
	DebugLogOfType("attract", "publishOscAlive", "uptimesecs", uptimesecs, "attract", attractMode)
	if sched.attractClient == nil {
		sched.attractClient = osc.NewClient("127.0.0.1", 3331)
	}
	msg := osc.NewMessage("/alive")
	msg.Append(float32(uptimesecs))
	msg.Append(attractMode)
	err := sched.attractClient.Send(msg)
	if err != nil {
		Warn("publishOscAlive", "err", err)
	}
}

func (sched *Scheduler) doAttractAction() {

	now := time.Now()
	dt := now.Sub(sched.lastAttractGestureTime)
	if sched.attractModeIsOn && dt > sched.attractGestureDuration {
		playerNames := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		player := playerNames[i]
		sched.lastAttractGestureTime = now

		cid := fmt.Sprintf("%d", time.Now().UnixNano())

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		go TheEngine().CursorManager.doCursorGesture(player, cid, x0, y0, z0, x1, y1, z1)
		sched.lastAttractGestureTime = now
	}

	sinceLast := now.Sub(sched.lastAttractPresetTime)
	/*
		presetdur := sched.attractPresetDuration
		Info("attractPreset check",
			"sincelast", sinceLast,
			"presetdur", presetdur,
			"now", now,
			"lastpresettime", sched.lastAttractPresetTime)
	*/
	if sched.attractPreset == "random" && sinceLast > sched.attractPresetDuration {
		TheEngine().Router.loadQuadPresetRand()
		sched.lastAttractPresetTime = now
	}
}

func (sched *Scheduler) advanceTransposeTo(newclick Clicks) {
	if sched.transposeAuto && sched.transposeNext < newclick {
		sched.transposeNext += (sched.transposeClicks * oneBeat)
		sched.transposeIndex = (sched.transposeIndex + 1) % len(sched.transposeValues)
		transposePitch := sched.transposeValues[sched.transposeIndex]
		DebugLogOfType("transpose", "advanceTransposeTo", "newclick", newclick, "transposePitch", transposePitch)
		/*
			for _, player := range TheEngine().Router.players {
				// player.clearDown()
				LogOfType("transpose","setting transposepitch in player","pad", player.padName, "transposePitch",transposePitch, "nactive",len(player.activeNotes))
				player.TransposePitch = transposePitch
			}
		*/
		sched.activePhrasesManager.terminateActiveNotes()
	}
}
