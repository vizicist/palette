package engine

import (
	"container/list"
	"fmt"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
)

type Event any

type Scheduler struct {
	schedList *list.List // of *SchedElements
	// schedMutex sync.RWMutex

	pendingNoteOffs *Phrase

	// now       time.Time
	// time0     time.Time
	lastClick Clicks
	cmdInput  chan any
}

type Command struct {
	Action string // e.g. "addmidi"
	Arg    any
}

type SchedElement struct {
	AtClick   Clicks
	layer     *Layer
	Value     any // ...SchedValue things
	triggered bool
}

type MidiSchedValue struct {
	msg midi.Message
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		schedList: list.New(),
		// pendingNoteOffs: NewPhrase(),
		// now:             time.Time{},
		// time0:           time.Time{},
		lastClick: -1,
		cmdInput:  make(chan any),
	}
	return s
}

type SchedulerElementCmd struct {
	element *SchedElement
}

// Start runs the scheduler and never returns
func (sched *Scheduler) Start() {

	LogInfo("Scheduler begins")

	// Wake up every 2 milliseconds and check looper events
	tick := time.NewTicker(2 * time.Millisecond)
	// sched.time0 = <-tick.C

	nonRealtime := false

	// By reading from tick.C, we wake up every 2 milliseconds
	for range tick.C {
		// sched.now = now
		uptimesecs := Uptime()

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

		sched.advanceClickTo(newclick)

		SetCurrentClick(newclick)

		sched.checkInput()

		ce := ClickEvent{Click: newclick, Uptime: uptimesecs}
		ThePluginManager().handleClickEvent(ce)
	}
	LogInfo("StartRealtime ends")
}

func (sched *Scheduler) checkInput() {

	select {
	case cmd := <-sched.cmdInput:
		LogInfo("Scheduler.cmdInput", "cmd", cmd)
		switch v := cmd.(type) {
		case SchedulerElementCmd:
			sched.scheduleElement(v.element)
		default:
			LogWarn("Unexpected type", "type", fmt.Sprintf("%T", v))
		}
	default:
	}
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// LogOfType("scheduler", "Scheduler.advanceClickTo", "toClick", toClick, "scheduler", sched)

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
		// sched.advancePendingNoteOffsByOneClick()
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
					LogWarn("SchedElement already triggered?")
				}
			default:
				msg := fmt.Sprintf("triggerItemsScheduleAt: unexpected Value type=%T", v)
				LogWarn(msg)
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
			LogWarn("Unexpected value in Phrase list")
			break
		}
		if pe.AtClick <= dclick {
			// Info("triggerPhraseElementAt", "click", clk, "dclick", dclick, "pe.AtClick", pe.AtClick)
			switch v := pe.Value.(type) {
			case *NoteOn:
				LogWarn("triggerPhraseElementsAt: NoteOn not handled")
				// SendToSynth(v)
			case *NoteOff:
				LogWarn("triggerPhraseElementsAt: NoteOff not handled")
				// SendToSynth(v)
			case *NoteFull:
				LogWarn("triggerPhraseElementsAt: NoteFull not handled")
				// noteon := NewNoteOn(v.Channel, v.Pitch, v.Velocity)
				// SendToSynth(noteon)
				// noteoff := NewNoteOff(v.Channel, v.Pitch, v.Velocity)
				// offClick := clk + pe.AtClick + v.Duration
				// newpe := &PhraseElement{AtClick: offClick, Value: noteoff}
				// sched.pendingNoteOffs.InsertElement(newpe)
			default:
				msg := fmt.Sprintf("triggerPhraseElementsAt: unexpected Value type=%T", v)
				LogWarn(msg)
			}
		}
	}
}

/*
// Time returns the current time
func (sched *Scheduler) time() time.Time {
	return time.Now()
}
*/

/*
func (sched *Scheduler) Uptime() float64 {
	return sched.now.Sub(sched.time0).Seconds()
}
*/

/*
// StartPhrase xxx
func (sched *Scheduler) AddActivePhraseAt(click Clicks, phrase *Phrase, sid string) {
	if phrase == nil {
		Warn("AddActivePhraseAt: Unexpected nil phrase")
		return
	}
	LogOfType("phrase", "StartPhrase", "sid", sid, "phrase", phrase)
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

	LogWarn("SendAllPendingNoteoffs needs work")
	/*
		var nexti *list.Element
		for i := sched.pendingNoteOffs.list.Front(); i != nil; i = nexti {
			nexti = i.Next()
			pe, ok := i.Value.(*PhraseElement)
			if !ok {
				LogWarn("Non-PhraseElement in activeNotes!?", "value", i.Value)
				continue

			}
			noff, ok := pe.Value.(*NoteOff)
			if !ok {
				LogWarn("Non-NoteOff in activeNotes!?", "value", i.Value)
				continue
			}
			SendToSynth(noff)
			sched.pendingNoteOffs.list.Remove(i)
		}
	*/
}

// AdvanceByOneClick xxx
func (sched *Scheduler) advancePendingNoteOffsByOneClick() {

	sched.pendingNoteOffs.rwmutex.Lock()
	defer sched.pendingNoteOffs.rwmutex.Unlock()

	LogWarn("advancePendingNoteoffs needs work")
	/*
		currentClick := CurrentClick()
		var nexti *list.Element
		for i := sched.pendingNoteOffs.list.Front(); i != nil; i = nexti {
			nexti = i.Next()
			pe := i.Value.(*PhraseElement)
			ntoff, ok := pe.Value.(*NoteOff)
			if !ok {
				LogWarn("Non NoteOff in pendingNoteOffs?")
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
	*/
}

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
	LogOfType("scheduler", "Scheduler.scheduleElement", "se", se, "click", schedClick)
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
