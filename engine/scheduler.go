package engine

import (
	"container/list"
	"fmt"
	"time"
)

type Event interface{}

type Scheduler struct {
	schedList *list.List // of *SchedElements
	// schedMutex sync.RWMutex

	pendingNoteOffs *Phrase

	now       time.Time
	time0     time.Time
	lastClick Clicks
	cmdInput  chan interface{}

	transposeAuto   bool
	transposeNext   Clicks
	transposeClicks Clicks // time between auto transpose changes
	transposeIndex  int    // current place in tranposeValues
	transposeValues []int

	lastProcessCheck float64
	processCheckSecs float64
	aliveSecs        float64
	lastAlive        float64
}

type Command struct {
	Action string // e.g. "addmidi"
	Arg    interface{}
}

type SchedElement struct {
	AtClick   Clicks
	Value     interface{} // currently just *Phrase values, but others should be possible
	triggered bool
}

func NewScheduler() *Scheduler {
	transposebeats := Clicks(ConfigIntWithDefault("transposebeats", 48))
	s := &Scheduler{
		schedList:       list.New(),
		pendingNoteOffs: NewPhrase(),
		now:             time.Time{},
		time0:           time.Time{},
		lastClick:       -1,
		cmdInput:        make(chan interface{}),
		transposeAuto:   ConfigBoolWithDefault("transposeauto", true),
		transposeNext:   transposebeats * OneBeat,
		transposeClicks: transposebeats,
		transposeIndex:  0,
		transposeValues: []int{0, -2, 3, -5},

		lastProcessCheck: 0,
		processCheckSecs: 0,
		aliveSecs:        0,
		lastAlive:        0,
	}
	return s
}

type AttractModeCmd struct {
	onoff bool
}

type SchedulerElementCmd struct {
	element *SchedElement
}

// Start runs the scheduler and never returns
func (sched *Scheduler) Start(onClick func(click ClickEvent)) {

	Info("Scheduler begins")

	// Wake up every 2 milliseconds and check looper events
	tick := time.NewTicker(2 * time.Millisecond)
	sched.time0 = <-tick.C

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	sched.processCheckSecs = float64(ConfigFloatWithDefault("processchecksecs", 60))

	sched.aliveSecs = float64(ConfigFloatWithDefault("alivesecs", 5))

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

		if newclick <= thisClick {
			// Info("SCHEDULER skipping to next loop, newclick is unchanged","newclick",newclick,"currentClick",currentClick)
			continue
		}
		ce := ClickEvent{Click: newclick, Uptime: uptimesecs}
		onClick(ce)

		TheRouter().taskManager.handleClickEvent(ce)
	}
	Info("StartRealtime ends")
}

func (sched *Scheduler) DefaultOnClick(ce ClickEvent) {

	newclick := ce.Click

	sched.advanceTransposeTo(newclick)
	sched.advanceClickTo(newclick)

	SetCurrentClick(newclick)

	/*
		uptimesecs := sched.Uptime()
		sinceLastAlive := uptimesecs - sched.lastAlive
		if sinceLastAlive > sched.aliveSecs {
			sched.publishOscAlive(uptimesecs)
			sched.lastAlive = uptimesecs
		}
	*/

	/*
		processCheckEnabled := sched.processCheckSecs > 0

		// At the beginning and then every processCheckSecs seconds
		// we check to see if necessary processes are still running
		firstTime := (sched.lastProcessCheck == 0)
		sinceLastProcessCheck := uptimesecs - sched.lastProcessCheck
		if processCheckEnabled && (firstTime || sinceLastProcessCheck > sched.processCheckSecs) {
			// Put it in background, so calling
			// tasklist or ps doesn't disrupt realtime
			// The sleep here is because if you kill something and
			// immediately check to see if it's running, it reports that
			// it's stil running.
			go func() {
				DebugLogOfType("scheduler", "Scheduler: checking processes")
				time.Sleep(2 * time.Second)
				TheEngine().ProcessManager.checkProcessesAndRestartIfNecessary()
			}()
			sched.lastProcessCheck = uptimesecs
		}
		if !processCheckEnabled && firstTime {
			Info("Process Checking is disabled.")
		}
	*/

	select {
	case cmd := <-sched.cmdInput:
		// Info("Realtime.Control", "cmd", cmd)
		switch v := cmd.(type) {
		/*
			case AttractModeCmd:
				onoff := v.onoff
				if sched.attractModeIsOn != onoff {
					sched.attractModeIsOn = onoff
					sched.lastAttractChange = sched.Uptime()
				}
		*/
		case SchedulerElementCmd:
			sched.scheduleElement(v.element)
		default:
			Warn("Unexpected type", "type", fmt.Sprintf("%T", v))
		}
	default:
	}
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// DebugLogOfType("scheduler", "Scheduler.advanceClickTo", "toClick", toClick, "scheduler", sched)

	// Don't let events get handled while we're advancing
	// XXX - this might not be needed if all communication/syncing
	// is done only from the scheduler loop
	TheRouter().inputEventMutex.Lock()
	defer func() {
		TheRouter().inputEventMutex.Unlock()
	}()

	doAutoCursorUp := false
	sched.lastClick += 1
	for clk := sched.lastClick; clk <= toClick; clk++ {
		sched.triggerItemsScheduledAt(clk)
		sched.advancePendingNoteOffsByOneClick()
		if doAutoCursorUp {
			TheRouter().cursorManager.autoCursorUp(time.Now())
		}
	}
	sched.lastClick = toClick
}

func (sched *Scheduler) triggerItemsScheduledAt(clk Clicks) {
	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		dclick := clk - se.AtClick
		if dclick >= 0 {
			switch v := se.Value.(type) {
			case *Phrase:
				if !se.triggered {
					se.triggered = true
					sched.triggerPhraseElementsAt(v, clk, dclick)
				} else {
					Warn("SchedElement already triggered?")
				}
			default:
				msg := fmt.Sprintf("triggerItemsScheduleAt: unexpected Value type=%T", v)
				Warn(msg)
			}

			// XXX - this is (maybe) where looping happens
			// by rescheduling things rather than removing
			sched.schedList.Remove(i)
		}
	}
}

func (sched *Scheduler) triggerPhraseElementsAt(phr *Phrase, clk Clicks, dclick Clicks) {
	for i := phr.list.Front(); i != nil; i = i.Next() {
		pe, ok := i.Value.(*PhraseElement)
		if !ok {
			Warn("Unexpected value in Phrase list")
			break
		}
		if pe.AtClick <= dclick {
			// Info("triggerPhraseElementAt", "click", clk, "dclick", dclick, "pe.AtClick", pe.AtClick)
			switch v := pe.Value.(type) {
			case *NoteOn:
				SendToSynth(v)
			case *NoteOff:
				SendToSynth(v)
			case *NoteFull:
				noteon := NewNoteOn(v.Channel, v.Pitch, v.Velocity)
				SendToSynth(noteon)
				noteoff := NewNoteOff(v.Channel, v.Pitch, v.Velocity)
				offClick := clk + pe.AtClick + v.Duration
				newpe := &PhraseElement{AtClick: offClick, Value: noteoff}
				sched.pendingNoteOffs.InsertElement(newpe)
			default:
				msg := fmt.Sprintf("triggerPhraseElementsAt: unexpected Value type=%T", v)
				Warn(msg)
			}
		}
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

/*
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

	sched.pendingNoteOffs.rwmutex.Lock()
	defer sched.pendingNoteOffs.rwmutex.Unlock()

	// Insert accoring to click
	// XXX - should be generic-ized
	i := sched.pendingNoteOffs.list.Front()
	if i == nil {
		sched.pendingNoteOffs.list.PushFront(newItem)
	} else if sched.pendingNoteOffs.list.Back().Value.(*ActivePhrase).AtClick <= newItem.AtClick {
		sched.pendingNoteOffs.list.PushBack(newItem)
	} else {
		for ; i != nil; i = i.Next() {
			ap := i.Value.(*ActivePhrase)
			if ap.AtClick > click {
				sched.pendingNoteOffs.list.InsertBefore(newItem, i)
			}
		}
	}
}

*/

// StopPhrase xxx
func (sched *Scheduler) SendAllPendingNoteoffs() {

	sched.pendingNoteOffs.rwmutex.Lock()
	defer sched.pendingNoteOffs.rwmutex.Unlock()

	var nexti *list.Element
	for i := sched.pendingNoteOffs.list.Front(); i != nil; i = nexti {
		nexti = i.Next()
		noff, ok := i.Value.(*NoteOff)
		if !ok {
			Warn("Non-NoteOff in activeNotes!?", "value", i.Value)
			continue
		}
		SendToSynth(noff)
		sched.pendingNoteOffs.list.Remove(i)
	}
}

// AdvanceByOneClick xxx
func (sched *Scheduler) advancePendingNoteOffsByOneClick() {

	sched.pendingNoteOffs.rwmutex.Lock()
	defer sched.pendingNoteOffs.rwmutex.Unlock()

	currentClick := CurrentClick()
	var nexti *list.Element
	for i := sched.pendingNoteOffs.list.Front(); i != nil; i = nexti {
		nexti = i.Next()
		pe := i.Value.(*PhraseElement)
		ntoff, ok := pe.Value.(*NoteOff)
		if !ok {
			Warn("Non NoteOff in pendingNoteOffs?")
			sched.pendingNoteOffs.list.Remove(i)
			continue
		}
		if pe.AtClick > currentClick {
			// Warn("Scheduler.advancePendingNoteOffsByOneClick: clickStart > currentClick?")
		} else {
			SendToSynth(ntoff)
			sched.pendingNoteOffs.list.Remove(i)
		}
	}
}

/*
func (sched *Scheduler) terminateActiveNotes() {

	sched.activeNotes.rwmutex.RLock()
	defer sched.activeNotes.rwmutex.RUnlock()

	for i := sched.activeNotes.list.Front(); i != nil; i = i.Next() {
		ap := i.Value.(*ActivePhrase)
		if ap != nil {
			ap.sendPendingNoteOffs(ap.SofarClick)
		} else {
			Warn("Hey, nil activeNotes entry")
		}
	}
}
*/

func (sched *Scheduler) ToString() string {
	s := "Scheduler{"
	for i := sched.schedList.Front(); i != nil; i = i.Next() {
		pe := i.Value.(*SchedElement)
		switch v := pe.Value.(type) {
		case *Phrase:
			phr := v
			s += fmt.Sprintf("(%d,%v)", pe.AtClick, phr)
		default:
			s += "(Unknown SchedElement Type)"
		}
	}
	s += "}"
	return s
}

func (sched *Scheduler) Format(f fmt.State, c rune) {
	s := sched.ToString()
	f.Write([]byte(s))
}

func (sched *Scheduler) scheduleElement(se *SchedElement) {
	schedClick := se.AtClick
	DebugLogOfType("scheduler", "Scheduler.scheduleElement", "se", se, "click", schedClick)
	// Insert newElement sorted by time
	i := sched.schedList.Front()
	if i == nil {
		// new list
		sched.schedList.PushFront(se)
	} else if sched.schedList.Back().Value.(*SchedElement).AtClick <= schedClick {
		// pe is later than all existing things
		sched.schedList.PushBack(se)
	} else {
		// use click to find place to insert
		for ; i != nil; i = i.Next() {
			if i.Value.(*SchedElement).AtClick > schedClick {
				sched.schedList.InsertBefore(se, i)
				break
			}
		}
	}
}

func (sched *Scheduler) advanceTransposeTo(newclick Clicks) {
	sched.transposeNext += (sched.transposeClicks * OneBeat)
	sched.transposeIndex = (sched.transposeIndex + 1) % len(sched.transposeValues)
	transposePitch := sched.transposeValues[sched.transposeIndex]
	DebugLogOfType("transpose", "advanceTransposeTo", "newclick", newclick, "transposePitch", transposePitch)
	/*
		fr _, player := range TheRouter().players {
			// player.clearDown()
			LogOfType("transpose""setting transposepitch in player","pad", player.padName, "transposePitch",transposePitch, "nactive",len(player.activeNotes))
			player.TransposePitch = transposePitch
		}
	*/
	sched.SendAllPendingNoteoffs()
}
