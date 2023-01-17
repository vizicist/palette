package engine

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

type Event any

type Scheduler struct {
	schedList *list.List // of *SchedElements
	mutex     sync.RWMutex
	// now       time.Time
	// time0     time.Time
	lastClick Clicks
	// cmdInput  chan any
}

type Command struct {
	Action string // e.g. "addmidi"
	Arg    any
}

type SchedElement struct {
	AtClick Clicks
	Value   any // ...SchedValue things
	// triggered bool
}

// type MidiSchedValue struct {
// 	msg midi.Message
// }

func NewScheduler() *Scheduler {
	s := &Scheduler{
		schedList: list.New(),
		lastClick: -1,
		// cmdInput:  make(chan any),
	}
	return s
}

// type SchedulerElementCmd struct {
// 	element *SchedElement
// }

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

		// sched.checkInput()

		ce := ClickEvent{Click: newclick, Uptime: uptimesecs}
		ThePluginManager().handleClickEvent(ce)
	}
	LogInfo("StartRealtime ends")
}

/*
func (sched *Scheduler) checkInput() {

	select {
	case cmd := <-sched.cmdInput:
		LogInfo("Scheduler.cmdInput", "cmd", cmd)
		switch v := cmd.(type) {
		case SchedulerElementCmd:
			sched.insertScheduleElement(v.element)
		default:
			LogWarn("Unexpected type", "type", fmt.Sprintf("%T", v))
		}
	default:
	}
}
*/

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
		sched.triggerItemsScheduledAtOrBefore(clk)
		// sched.advancePendingNoteOffsByOneClick()
		if doAutoCursorUp {
			TheRouter().cursorManager.autoCursorUp(time.Now())
		}
	}
	sched.lastClick = toClick
}

func (sched *Scheduler) triggerItemsScheduledAtOrBefore(clk Clicks) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		dclick := clk - se.AtClick
		if dclick >= 0 {
			switch v := se.Value.(type) {
			/*
				case *Phrase:
					if !se.triggered {
						se.triggered = true
						sched.triggerPhraseElementsAt(v, clk, dclick)
					} else {
						LogWarn("SchedElement already triggered?")
					}
			*/

			case *NoteOn:
				LogOfType("scheduler", "triggerItemsScheduleAt: NoteOn", "note", v.String())
				v.Synth.SendNoteToMidiOutput(v)

			case *NoteOff:
				LogOfType("scheduler", "triggerItemsScheduleAt: NoteOff", "note", v.String())
				v.Synth.SendNoteToMidiOutput(v)

			default:
				LogError(fmt.Errorf("triggerItemsScheduleAt: unhandled Value type=%T", v))
			}

			// XXX - this is (maybe) where looping happens
			// by rescheduling things rather than removing
			sched.schedList.Remove(i)

			// LogInfo("After Removing from schedList", "i", i, "Len", sched.schedList.Len())
		}
	}
}

/*
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
*/

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

func (sched *Scheduler) ToString() string {

	// sched.mutex.Lock()
	// defer sched.mutex.Unlock()

	s := "Scheduler{"
	for i := sched.schedList.Front(); i != nil; i = i.Next() {
		pe := i.Value.(*SchedElement)
		switch v := pe.Value.(type) {
		/*
			case *Phrase:
				phr := v
				s += fmt.Sprintf("(%d,%v)", pe.AtClick, phr)
		*/
		case *NoteOn:
			s += fmt.Sprintf("(%d,%s)", pe.AtClick, v.String())
		case *NoteOff:
			s += fmt.Sprintf("(%d,%s)", pe.AtClick, v.String())
		default:
			s += fmt.Sprintf("(Unknown SchedElement Type=%T)", v)
		}
	}
	s += "}"
	return s
}

func (sched *Scheduler) Format(f fmt.State, c rune) {
	s := sched.ToString()
	f.Write([]byte(s))
}

func (sched *Scheduler) insertScheduleElement(se *SchedElement) {

	sched.mutex.Lock()

	schedClick := se.AtClick
	LogOfType("scheduler", "Scheduler.insertScheduleElement", "value", se.Value, "click", se.AtClick, "beforelen", sched.schedList.Len())
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

	LogOfType("scheduler", "Scheduler.insertScheduleElement", "value", se.Value, "click", se.AtClick, "schedafter", sched.ToString())

	sched.mutex.Unlock()
}
