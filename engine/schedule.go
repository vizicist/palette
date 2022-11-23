package engine

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Scheduler struct {
	schedule     *Phrase
	activePhrase *Phrase

	// activePhraseManager *ActivePhrasesManager

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

// type SchedItem struct {
// 	clickStart Clicks
// 	ID         string
// 	Value      interface{}
// }

func NewScheduler() *Scheduler {
	transposebeats := Clicks(ConfigIntWithDefault("transposebeats", 48))
	s := &Scheduler{
		schedule:               NewPhrase(),
		activePhrase:           NewPhrase(),
		now:                    time.Time{},
		time0:                  time.Time{},
		lastClick:              -1,
		cmdInput:               make(chan interface{}),
		transposeAuto:          ConfigBoolWithDefault("transposeauto", true),
		transposeNext:          transposebeats * OneBeat,
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

type SchedulePhraseElementCmd struct {
	element *PhraseElement
	click   Clicks
}

type ScheduleBytesCmd struct {
	bytes []byte
	click Clicks
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

		// XXX - should lock from here?

		thisClick := CurrentClick()
		var newclick Clicks
		if nonRealtime {
			newclick = thisClick + 1
		} else {
			newclick = Seconds2Clicks(uptimesecs)
		}
		SetCurrentMilli(int64(uptimesecs * 1000.0))

		// XXX - should unlock here?

		// Info("SCHEDULER TOP OF LOOP", "currentClick", currentClick, "newclick", newclick)

		if newclick <= thisClick {
			// Info("SCHEDULER skipping to next loop, newclick is unchanged","newclick",newclick,"currentClick",currentClick)
			continue
		}

		sched.advanceTransposeTo(newclick)
		sched.advanceClickTo(newclick)

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

		// At the beginning and then every processCheckSecs seconds
		// we check to see if necessary processes are still running
		firstTime := (lastProcessCheck == 0)
		sinceLastProcessCheck := uptimesecs - lastProcessCheck
		processCheckEnabled := sched.processCheckSecs > 0
		if processCheckEnabled && (firstTime || sinceLastProcessCheck > sched.processCheckSecs) {
			// Put it in background, so calling
			// tasklist or ps doesn't disrupt realtime
			DebugLogOfType("schedule", "StartRealtime: checking processes")
			go TheEngine().ProcessManager.checkProcessesAndRestartIfNecessary()
		}
		if !processCheckEnabled && firstTime {
			Info("Process Checking is disabled.")
		}
		lastProcessCheck = uptimesecs

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
				sched.schedulePhraseAt(v.phrase, v.click)
				/*
					case ScheduleBytesCmd:
						nt := NewBytes(v.bytes)
						phr := NewPhrase().InsertNote(nt)
						sched.schedulePhraseAt(phr, v.click)
				*/
			default:
				Warn("Unexpected type", "type", fmt.Sprintf("%T", v))
			}
		default:
		}
	}
	Info("StartRealtime ends")
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	DebugLogOfType("schedule", "Scheduler.advanceClickTo", "toClick", toClick, "schedule", sched)

	// Don't let events get handled while we're advancing
	// XXX - this might not be needed if all communication/syncing
	// is done only from the schedule loop
	TheRouter().inputEventMutex.Lock()
	defer func() {
		TheRouter().inputEventMutex.Unlock()
	}()

	doAutoCursorUp := false
	sched.lastClick += 1
	for clk := sched.lastClick; clk <= toClick; clk++ {
		sched.triggerItemsScheduledAt(clk)
		sched.advanceActivePhrasesByOneClick()
		if doAutoCursorUp {
			TheRouter().CursorManager.autoCursorUp(time.Now())
		}
	}
	sched.lastClick = toClick
}

func (sched *Scheduler) triggerItemsScheduledAt(clk Clicks) {
	for i := sched.schedule.list.Front(); i != nil; {
		nexti := i.Next()
		pe := i.Value.(*PhraseElement)
		if pe.AtClick <= clk {
			switch v := pe.Value.(type) {
			case *Phrase:
				phrase := v
				sched.AddActivePhraseAt(pe.AtClick, phrase, "")
			}

			// XXX - this is (maybe) where looping happens
			// by rescheduling things rather than removing
			sched.schedule.list.Remove(i)
		}
		i = nexti
	}
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
		sched.attractClient = osc.NewClient(LocalAddress, AliveOutputPort)
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

		go TheRouter().CursorManager.generateCursorGestureesture(player, cid, x0, y0, z0, x1, y1, z1)
		sched.lastAttractGestureTime = now
	}

	dp := now.Sub(sched.lastAttractPresetTime)
	if sched.attractPreset == "random" && dp > TheEngine().Scheduler.attractPresetDuration {
		TheRouter().loadQuadPresetRand()
		sched.lastAttractPresetTime = now
	}
}

// StartPhrase xxx
func (sched *Scheduler) AddActivePhraseAt(click Clicks, phrase *Phrase, sid string) {
	if phrase == nil {
		Warn("AddActivePhraseAt: Unexpected nil phrase")
		return
	}
	DebugLogOfType("phrase", "StartPhrase", "sid", sid, "phrase", phrase)
	newItem := &ActivePhrase{
		phrase:          phrase,
		AtClick:         click,
		SofarClick:      0,
		pendingNoteOffs: NewPhrase(),
	}

	sched.activePhrase.rwmutex.Lock()
	defer sched.activePhrase.rwmutex.Unlock()

	// Insert accoring to click
	// XXX - should be generic-ized
	i := sched.activePhrase.list.Front()
	if i == nil {
		sched.activePhrase.list.PushFront(newItem)
	} else if sched.activePhrase.list.Back().Value.(*ActivePhrase).AtClick <= newItem.AtClick {
		sched.activePhrase.list.PushBack(newItem)
	} else {
		for ; i != nil; i = i.Next() {
			ap := i.Value.(*ActivePhrase)
			if ap.AtClick > click {
				sched.activePhrase.list.InsertBefore(newItem, i)
			}
		}
	}
}

// StopPhrase xxx
func (sched *Scheduler) StopPhrase(source string, active *ActivePhrase) {
	DebugLogOfType("phrase", "StopPhrase", "source", source)
	for i := sched.activePhrase.list.Front(); i != nil; i = i.Next() {
		ap := i.Value.(*ActivePhrase)
		if ap.phrase.Source == source {
			readyToDelete := ap.sendPendingNoteOffs(MaxClicks)
			if !readyToDelete {
				Warn("StopPhrase, why isn't ap ready to delete?", "source", source)
			} else {
				sched.activePhrase.list.Remove(i)
			}
		}
	}
}

// AdvanceByOneClick xxx
func (sched *Scheduler) advanceActivePhrasesByOneClick() {

	sched.activePhrase.rwmutex.Lock()
	defer sched.activePhrase.rwmutex.Unlock()

	currentClick := CurrentClick()
	for i := sched.activePhrase.list.Front(); i != nil; i = i.Next() {
		ap := i.Value.(*ActivePhrase)
		if ap.phrase == nil {
			Warn("advanceactivePhrase, unexpected phrase is nil")
			sched.activePhrase.list.Remove(i)
		} else if ap.AtClick > currentClick {
			Warn("Scheduler.AdvanceByOneClick: clickStart > currentClick?")
		} else {
			if ap.AdvanceByOneClick() {
				sched.activePhrase.list.Remove(i)
			}
		}
	}
}

func (sched *Scheduler) terminateActiveNotes() {

	sched.activePhrase.rwmutex.RLock()
	defer sched.activePhrase.rwmutex.RUnlock()

	for i := sched.activePhrase.list.Front(); i != nil; i = i.Next() {
		// for id, a := range sched.activePhrase {
		ap := i.Value.(*ActivePhrase)
		if ap != nil {
			ap.sendPendingNoteOffs(ap.SofarClick)
		} else {
			Warn("Hey, nil activeNotes entry")
		}
	}
}

func (sched *Scheduler) ToString() string {
	s := "Schedule{"
	for i := sched.schedule.list.Front(); i != nil; i = i.Next() {
		pe := i.Value.(*PhraseElement)
		switch v := pe.Value.(type) {
		case *Phrase:
			phr := v
			s += fmt.Sprintf("(%d,%s)", pe.AtClick, phr)
		default:
			s += "(Unknown Sched Type)"
		}
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
	DebugLogOfType("schedule", "Scheduler.SchedulePhraseAt", "phrase", phr, "click", click)
	newElement := &PhraseElement{
		AtClick: click,
		Value:   phr,
	}

	// Insert newElement sorted by time
	// XXX - could be generic-ized with identical code for activePhrase
	i := schedule.list.Front()
	if i == nil {
		// new list
		schedule.list.PushFront(newElement)
	} else if schedule.list.Back().Value.(*PhraseElement).AtClick <= newElement.AtClick {
		// newItem is later than all existing things
		schedule.list.PushBack(newElement)
	} else {
		// use click to find place to insert
		for ; i != nil; i = i.Next() {
			si := i.Value.(*PhraseElement)
			if si.AtClick > click {
				schedule.list.InsertBefore(newElement, i)
			}
		}
	}
	return id
}

func (sched *Scheduler) advanceTransposeTo(newclick Clicks) {
	if sched.transposeAuto && sched.transposeNext < newclick {
		sched.transposeNext += (sched.transposeClicks * OneBeat)
		sched.transposeIndex = (sched.transposeIndex + 1) % len(sched.transposeValues)
		transposePitch := sched.transposeValues[sched.transposeIndex]
		DebugLogOfType("transpose", "advanceTransposeTo", "newclick", newclick, "transposePitch", transposePitch)
		/*
			for _, player := range TheRouter().players {
				// player.clearDown()
				LogOfType("transpose","setting transposepitch in player","pad", player.padName, "transposePitch",transposePitch, "nactive",len(player.activeNotes))
				player.TransposePitch = transposePitch
			}
		*/
		sched.terminateActiveNotes()
	}
}
