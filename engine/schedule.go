package engine

import (
	"container/list"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Scheduler struct {
	mutex         sync.RWMutex
	schedule      *list.List // time-ordered by Clicks
	activePhrases *list.List // time-ordered by Clicks
	// mutex    sync.RWMutex
	nextSid int

	// activePhrasesManager *ActivePhrasesManager

	now       time.Time
	time0     time.Time
	lastClick Clicks
	cmdInput  chan interface{}

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

type Command struct {
	Action string // e.g. "addmidi"
	Arg    interface{}
}

type SchedItem struct {
	clickStart Clicks
	ID         string
	phrase     *Phrase
}

func NewScheduler() *Scheduler {
	transposebeats := Clicks(ConfigIntWithDefault("transposebeats", 48))
	s := &Scheduler{
		schedule:               list.New(),
		activePhrases:          list.New(),
		nextSid:                0,
		now:                    time.Time{},
		time0:                  time.Time{},
		lastClick:              -1,
		cmdInput:               make(chan interface{}),
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

type AttractModeCmd struct {
	onoff bool
}

type SchedulePhraseCmd struct {
	phrase *Phrase
	click  Clicks
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

	nonRealtime := false

	// By reading from tick.C, we wake up every 2 milliseconds
	for now := range tick.C {
		sched.now = now
		uptimesecs := sched.Uptime()
		currentClick := CurrentClick()
		var newclick Clicks
		if nonRealtime {
			newclick = currentClick + 1
		} else {
			newclick = Seconds2Clicks(uptimesecs)
		}
		SetCurrentMilli(int64(uptimesecs * 1000.0))

		// Info("SCHEDULER TOP OF LOOP", "currentClick", currentClick, "newclick", newclick)

		if newclick <= currentClick {
			// Info("SCHEDULER skipping to next loop, newclick is unchanged","newclick",newclick,"currentClick",currentClick)
			continue
		}

		sched.advanceTransposeTo(newclick)
		sched.advanceClickTo(currentClick)
		SetCurrentClick(newclick)

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
					sched.cmdInput <- Command{"attractmode", true}
				}()
				// sched.SetAttractMode(true)
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
		case cmd := <-sched.cmdInput:
			// Info("Realtime.Control", "cmd", cmd)
			switch v := cmd.(type) {
			case AttractModeCmd:
				onoff := v.onoff
				if sched.attractModeIsOn != onoff {
					sched.attractModeIsOn = onoff
					sched.lastAttractChange = sched.Uptime()
				}
			case SchedulePhraseCmd:
				phr := v.phrase
				click := v.click
				sched.schedulePhraseAt(phr, click)
				Info("SchedulePhraseCmd sched is now", "schedule", sched.ToString())
			}
		default:
		}
	}
	Info("StartRealtime ends")
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// Info("Scheduler.advanceClickTo", "toClick", toClick, "schedule", sched.ToString())

	// Don't let events get handled while we're advancing
	// XXX - this might not be needed if all communication/syncing
	// is done only from the schedule loop
	TheEngine().Router.eventMutex.Lock()
	defer func() {
		TheEngine().Router.eventMutex.Unlock()
	}()

	// if (toClick - sched.lastClick) != 1 {
	// 	Warn("Hey, does this happen?", "toClick", toClick, "lastClick", sched.lastClick)
	// }
	// Info("sched.advanceClickTo")

	doAutoCursorUp := false
	for clk := sched.lastClick; clk < toClick; clk++ {
		// See if anything scheduled items are due
		for i := sched.schedule.Front(); i != nil; {
			nexti := i.Next()
			si := i.Value.(*SchedItem)
			if si.clickStart <= clk {
				sched.AddActivePhraseAt(si.clickStart, si.phrase, si.ID)

				// XXX - this is (maybe) where looping happens
				// by rescheduling things rather than removing
				sched.schedule.Remove(i)

			}
			i = nexti
		}

		sched.advanceActivePhrasesByOneClick()

		if doAutoCursorUp {
			TheEngine().CursorManager.autoCursorUp(time.Now())
		}
	}
	// Remove old things from the schedule
	for i := sched.schedule.Front(); i != nil; i = i.Next() {
		si := i.Value.(*SchedItem)
		if si.clickStart < toClick {
			sched.schedule.Remove(i)
		}
	}
	sched.lastClick = toClick
}

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

	dp := now.Sub(sched.lastAttractPresetTime)
	if sched.attractPreset == "random" && dp > TheEngine().Scheduler.attractPresetDuration {
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
		sched.terminateActiveNotes()
	}
}

// StartPhrase xxx
func (sched *Scheduler) AddActivePhraseAt(click Clicks, phrase *Phrase, sid string) {
	if phrase == nil {
		Warn("AddActivePhraseAt: Unexpected nil phrase")
		return
	}
	DebugLogOfType("phrase", "StartPhrase", "sid", sid)
	newItem := &ActivePhrase{
		phrase:          phrase,
		clickStart:      click,
		clickSoFar:      0,
		nextnote:        phrase.firstnote,
		pendingNoteOffs: NewPhrase(),
		sid:             sid,
	}
	activePhrases := sched.activePhrases

	// Insert accoring to click
	// XXX - should be generic-ized
	i := activePhrases.Front()
	if i == nil {
		activePhrases.PushFront(newItem)
	} else if activePhrases.Back().Value.(*ActivePhrase).clickStart <= newItem.clickStart {
		activePhrases.PushBack(newItem)
	} else {
		for ; i != nil; i = i.Next() {
			ap := i.Value.(*ActivePhrase)
			if ap.clickStart > click {
				activePhrases.InsertBefore(newItem, i)
			}
		}
	}

}

// StopPhrase xxx
func (sched *Scheduler) StopPhrase(sid string, active *ActivePhrase) {
	DebugLogOfType("phrase", "StopPhrase", "sid", sid)
	for i := sched.activePhrases.Front(); i != nil; i = i.Next() {
		ap := i.Value.(*ActivePhrase)
		if ap.sid == sid {
			readyToDelete := ap.sendPendingNoteOffs(MaxClicks)
			if !readyToDelete {
				Warn("StopPhrase, why isn't ap ready to delete?", "sid", sid)
			} else {
				sched.activePhrases.Remove(i)
			}
		}
	}
}

// AdvanceByOneClick xxx
func (sched *Scheduler) advanceActivePhrasesByOneClick() {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	currentClick := CurrentClick()
	for i := sched.activePhrases.Front(); i != nil; i = i.Next() {
		ap := i.Value.(*ActivePhrase)
		if ap.phrase == nil {
			Warn("advanceactivePhrases, unexpected phrase is nil", "sid", ap.sid)
			sched.activePhrases.Remove(i)
		} else if ap.clickStart > currentClick {
			Warn("Scheduler.AdvanceByOneClick: clickStart > currentClick?")
		} else {
			if ap.AdvanceByOneClick() {
				sched.activePhrases.Remove(i)
			}
		}
	}
}

func (sched *Scheduler) terminateActiveNotes() {
	sched.mutex.RLock()
	defer sched.mutex.RUnlock()

	for i := sched.activePhrases.Front(); i != nil; i = i.Next() {
		// for id, a := range sched.activePhrases {
		ap := i.Value.(*ActivePhrase)
		if ap != nil {
			ap.sendPendingNoteOffs(ap.clickSoFar)
		} else {
			Warn("Hey, nil activeNotes entry", "sid", ap.sid)
		}
	}
}

func (sched *Scheduler) ToString() string {
	s := "Schedule{"
	for i := sched.schedule.Front(); i != nil; i = i.Next() {
		si := i.Value.(*SchedItem)
		s += fmt.Sprintf("(%d,%s,%s)", si.clickStart, si.ID, si.phrase)
	}
	s += "}"
	return s
}

func (sched *Scheduler) Format(f fmt.State, c rune) {
	s := sched.ToString()
	f.Write([]byte(s))
}

func (sched *Scheduler) schedulePhraseAt(phr *Phrase, click Clicks) (id string) {
	schedule := sched.schedule
	Info("Scheduler.SchedulePhraseAt", "phrase", phr.ToString(), "click", click)
	id = fmt.Sprintf("%d", sched.nextSid)
	sched.nextSid += 1
	newItem := &SchedItem{
		clickStart: click,
		phrase:     phr,
		ID:         id,
	}

	// Insert newItem sorted by time
	// XXX - could be generic-ized with identical code for activePhrases
	i := schedule.Front()
	if i == nil {
		// new list
		schedule.PushFront(newItem)
	} else if schedule.Back().Value.(*SchedItem).clickStart <= newItem.clickStart {
		// newItem is later than all existing things
		schedule.PushBack(newItem)
	} else {
		// use click to find place to insert
		for ; i != nil; i = i.Next() {
			si := i.Value.(*SchedItem)
			if si.clickStart > click {
				schedule.InsertBefore(newItem, i)
			}
		}
	}
	return id
}
