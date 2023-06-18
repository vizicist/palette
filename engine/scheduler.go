package engine

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
)

var TheScheduler *Scheduler

type Event any

type Scheduler struct {
	mutex            sync.RWMutex
	schedList        *list.List // of *SchedElements
	lastClick        Clicks
	pendingMutex     sync.RWMutex
	pendingScheduled []*SchedElement
}

type Command struct {
	Action string // e.g. "addmidi"
	Arg    any
}

type SchedElement struct {
	// patch             *Patch
	AtClick *atomic.Int64
	Tag     string
	Value   any
	// loopIt            bool
	// loopLengthInBeats int
	// loopFade          float32
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		schedList:        list.New(),
		lastClick:        -1,
		pendingScheduled: nil,
	}
	InitializeClicksPerSecond(defaultClicksPerSecond)
	return s
}

func NewSchedElement(atclick Clicks, tag string, value any) *SchedElement {
	se := &SchedElement{
		AtClick: &atomic.Int64{},
		Tag:    tag,
		Value:   value,
	}
	se.SetClick(atclick)
	return se
}

func (se *SchedElement) SetClick(click Clicks) {
	if se.AtClick == nil {
		LogWarn("Hey, click is null in SchedElement.SetClick?")
		se.AtClick = &atomic.Int64{}
	}
	se.AtClick.Store(int64(click))
}
func (se *SchedElement) GetClick() Clicks {
	if se.AtClick == nil {
		LogWarn("Hey, click is null in SchedElement.SetClick?")
		se.AtClick = &atomic.Int64{}
	}
	return Clicks(se.AtClick.Load())
}

func ScheduleAt(atClick Clicks, tag string, value any) {
	ce, ok := value.(CursorEvent)
	if ok && ce.Gid == 0 {
		LogWarn("ScheduleAt Gid is 0", "ce", ce)
	}
	se := NewSchedElement(atClick, tag, value)
	TheScheduler.savePendingSchedEvent(se)
}

func (sched *Scheduler) savePendingSchedEvent(se *SchedElement) {

	sched.pendingMutex.Lock()
	defer sched.pendingMutex.Unlock()

	sched.pendingScheduled = append(sched.pendingScheduled, se)

	// LogInfo("savePendingSchedEvent", "se", se, "value", se.Value)

	// ss := fmt.Sprintf("%v",se.Value)
	// if strings.Contains(ss,"NoteOff") {
	// 	LogInfo("NoteOff in savePendingSchedEvent","se",se,"value",se.Value)
	// }
}

func (sched *Scheduler) handlePendingSchedEvents() {
	sched.pendingMutex.Lock()
	for _, se := range sched.pendingScheduled {
		TheScheduler.insertScheduleElement(se)
	}
	sched.pendingScheduled = nil
	sched.pendingMutex.Unlock()
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
		TheEngine.advanceTransposeTo(newclick)

		SetCurrentClick(newclick)

		sched.handlePendingSchedEvents()

		TheProcessManager.checkProcess()
		TheAttractManager.checkAttract()
	}
	LogInfo("StartRealtime ends")
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

	doAutoCursorUp := true
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

func (sched *Scheduler) DeleteEventsWhoseGidIs(gid int) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.Gid == gid {
			sched.schedList.Remove(i)
			// keep going, there will be lots of them
		}
	}
}

func (sched *Scheduler) DeleteEventsWithTag(tag string) {

	//// sched.ToString locks the mutex, to don't do it inside of the lock here
	// LogInfo("DeleteEventsForGidPrefix BEFORE", "prefix", prefix, "sched", sched.ToString())

	sched.mutex.Lock()

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.Tag == tag {
			sched.schedList.Remove(i)
			// keep going, there will be lots of them
		}
	}

	sched.mutex.Unlock()

	//// sched.ToString locks the mutex, to do it outside of the lock here
	// LogInfo("DeleteEventsForGidPrefix AFTER", "prefix", prefix, "sched", sched.ToString())
}

func (sched *Scheduler) triggerItemsScheduledAtOrBefore(thisClick Clicks) {

	sched.mutex.Lock()

	tobeExecuted := []CursorEvent{}

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {

		nexti = i.Next()
		se := i.Value.(*SchedElement)

		// too early?
		if (se.GetClick() - thisClick) > 0 {
			// XXX - should this continue be a break?  If the list is sorted by time, I think so!
			continue
		}

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
			ce := v
			if ce.Gid == 0 {
				LogWarn("Hey, Gid of CursorEvent is 0?")
			}
			if v.Ddu != "clear" && v.Tag == "" {
				LogWarn("Hey, Tag of CursorEvent is empty?")
			}
			// The Click in the CursorEvent is the click at which the event was scheduled,
			// which might be before clk
			ce.SetClick(se.GetClick())
			// delay the actual execution till the end of this routine
			tobeExecuted = append(tobeExecuted, ce)

		default:
			LogIfError(fmt.Errorf("triggerItemsScheduleAt: unhandled Value type=%T", v))
		}

		// This is where
		sched.schedList.Remove(i)
		// LogInfo("After Removing from schedList", "i", i, "Len", sched.schedList.Len())
	}

	sched.mutex.Unlock()

	for _, ce := range tobeExecuted {
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
		case CursorEvent:
			s += fmt.Sprintf("(%d,%v)", v.Click, v)
		default:
			s += fmt.Sprintf("(Unknown Type=%T)", v)
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
	defer sched.mutex.Unlock()

	switch v := (se.Value).(type) {
	case *NoteOn:
	case *NoteOff:
	case CursorEvent:
		if v.Ddu != "clear" && v.Gid == 0 {
			LogWarn("insertScheduleElement CursorEvent Gid is empty", "v", v)
		}
		// LogInfo("insertScheduleElement CursorEvent", "v", v)
		if v.Click == nil {
			LogWarn("insertScheduleElement CursorEvent AtClick is nil?", "v", v)
		}
	}
	schedClick := se.GetClick()
	LogOfType("scheduler", "Scheduler.insertScheduleElement", "value", se.Value, "click", se.AtClick, "beforelen", sched.schedList.Len())
	// Insert newElement sorted by time
	i := sched.schedList.Front()
	if i == nil {
		// new list
		sched.schedList.PushFront(se)
		// LogInfo("Adding SchedElement to front", "se", se)
	} else if sched.schedList.Back().Value.(*SchedElement).GetClick() <= schedClick {
		// pe is later than all existing things
		sched.schedList.PushBack(se)
		// LogInfo("Adding SchedElement to back", "se", se)
	} else {
		// use click to find place to insert
		for ; i != nil; i = i.Next() {
			if i.Value.(*SchedElement).GetClick() > schedClick {
				sched.schedList.InsertBefore(se, i)
				// LogInfo("Adding SchedElement to middle", "se", se)
				break
			}
		}
	}

	// LogOfType("scheduler", "Scheduler.insertScheduleElement", "value", se.Value, "click", se.AtClick, "schedafter", sched.ToString())

}
