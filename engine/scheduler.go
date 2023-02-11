package engine

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
)

var TheScheduler *Scheduler

type Event any

type Scheduler struct {
	schedList *list.List // of *SchedElements
	mutex     sync.RWMutex
	nextSeq   int
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
	patch   *Patch
	AtClick Clicks
	Value   any // ...SchedValue things
	// triggered bool
	loopIt            bool
	loopLengthInBeats int
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		schedList: list.New(),
		lastClick: -1,
		// cmdInput:  make(chan any),
	}
	InitializeClicksPerSecond(defaultClicksPerSecond)
	return s
}

func NewSchedElement(patch *Patch, atclick Clicks, value any) *SchedElement {
	// patch might be nil
	return &SchedElement{
		patch:             patch,
		AtClick:           atclick,
		Value:             value,
		loopIt:            false,
		loopLengthInBeats: 0,
	}
}

func ScheduleAt(patch *Patch, atClick Clicks, value any) {
	se := NewSchedElement(patch, atClick, value)
	TheScheduler.insertScheduleElement(se)
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
		TheMidiIO.advanceTransposeTo(newclick)

		SetCurrentClick(newclick)

		TheProcessManager.checkProcess()
		TheAttractManager.checkAttract()
	}
	LogInfo("StartRealtime ends")
}

func (sched *Scheduler) uniqueNum() int {
	sched.nextSeq++
	return sched.nextSeq
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// LogOfType("scheduler", "Scheduler.advanceClickTo", "toClick", toClick, "scheduler", sched)

	// Don't let events get handled while we're advancing
	// XXX - this might not be needed if all communication/syncing
	// is done only from the scheduler loop
	TheRouter.inputEventMutex.Lock()
	defer func() {
		TheRouter.inputEventMutex.Unlock()
	}()

	doAutoCursorUp := false
	sched.lastClick += 1
	for clk := sched.lastClick; clk <= toClick; clk++ {
		sched.triggerItemsScheduledAtOrBefore(clk)
		// sched.advancePendingNoteOffsByOneClick()
		if doAutoCursorUp {
			TheCursorManager.autoCursorUp(time.Now())
		}
	}
	sched.lastClick = toClick
}

func (sched *Scheduler) triggerItemsScheduledAtOrBefore(clk Clicks) {

	sched.mutex.Lock()

	// we don't handle CursorEvents while we're looping through the schedList,
	// since handling them may in some cases change the schedList in ways we can't predict.
	cursorEventsToExecute := []CursorEvent{}
	schedElementsToLoop := []*SchedElement{}

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

			case midi.Message:
				LogInfo("triggerItemsScheduleAt: should handle midi.Message?", "msg", v)

			case CursorEvent:
				if v.Cid == "" {
					LogWarn("Hey, Cid of CursorEvent is empty?")
				}
				cursorEventsToExecute = append(cursorEventsToExecute, v)

			default:
				LogError(fmt.Errorf("triggerItemsScheduleAt: unhandled Value type=%T", v))
			}

			// this is where looping happens
			// by rescheduling things (removing and then adding)
			if se.loopIt {
				schedElementsToLoop = append(schedElementsToLoop, se)
			}

			sched.schedList.Remove(i)
			// LogInfo("After Removing from schedList", "i", i, "Len", sched.schedList.Len())
		}
	}

	sched.mutex.Unlock()

	// Schedule the looped elements at their new time.
	// I *think* it's important to do this
	// before actually executing the CursorEvent.
	// e.g. if the CursorEvent is an "up", we probably
	// don't want to remove the Active cursor until we've
	// rescheduled its CursorEvent.

	for _, se := range schedElementsToLoop {

		// The SchedElement in the schedElementsToLoop list
		// has already been removed from the schedList,
		// se we can re-use it for the looped element.
		se.AtClick += OneBeat * Clicks(se.loopLengthInBeats)
		ce, ok := se.Value.(CursorEvent)
		if !ok {
			LogWarn("Hey, unexpected non-CursorEvent")
			continue
		}
		// The looped CursorEvents should have unique cid values.
		ce.Cid = TheCursorManager.LoopedCidFor(ce)
		if ce.Cid == "" {
			LogWarn("Unable to get LoopedCidFor", "ce", ce)
			continue
		}
		se.Value = ce
		TheScheduler.insertScheduleElement(se)
	}

	for _, ce := range cursorEventsToExecute {
		TheEngine.sendToOscClients(CursorToOscMsg(ce))
		TheCursorManager.ExecuteCursorEvent(ce)
	}
}

func (sched *Scheduler) ToString() string {

	sched.mutex.RLock()
	defer sched.mutex.RUnlock()

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

func LoopingIsSupported(value any) bool {
	// Currently, looping is only supported
	// for CursorEvent types
	_, ok := value.(CursorEvent)
	return ok
}

func FillInLooping(se *SchedElement) {
	if se.patch != nil {
		loopIt := se.patch.GetBool("misc.looping_on")
		loopBeats := se.patch.GetInt("misc.looping_beats")
		if loopIt && LoopingIsSupported(se.Value) {
			if loopBeats <= 0 {
				// sanity check
				LogWarn("loopBeats <= 0?")
				loopBeats = 8
			}
			se.loopIt = loopIt
			se.loopLengthInBeats = loopBeats
		} else {
			se.loopIt = false
		}
	} else {
		LogWarn("Hey, should patch in SchedElement be nil?")
	}
}

func (sched *Scheduler) insertScheduleElement(se *SchedElement) {

	FillInLooping(se)

	sched.mutex.Lock()

	ce, ok := se.Value.(CursorEvent)
	if ok && ce.Cid == "" {
		LogWarn("Hey, Cid of CursorEvent is empty?")
	}

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

	sched.mutex.Unlock()
	// Don't log before mutex.Unlock()
	LogOfType("scheduler", "Scheduler.insertScheduleElement", "value", se.Value, "click", se.AtClick, "schedafter", sched.ToString())

}
