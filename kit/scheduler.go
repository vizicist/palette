package kit

import (
	"container/list"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
)

var theScheduler *Scheduler

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
	AAtClick Clicks
	Tag      string
	Value    any
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
	return s
}

func NewSchedElement(atclick Clicks, tag string, value any) *SchedElement {
	se := &SchedElement{
		AAtClick: atclick,
		Tag:      tag,
		Value:    value,
	}
	se.SetClick(atclick)
	return se
}

// SetClick - NOTE: a pointer is used so se.SetClick() can modify the value
func (se *SchedElement) SetClick(click Clicks) {
	se.AAtClick = click
}

func (se SchedElement) GetClick() Clicks {
	return se.AAtClick
}

func ScheduleAt(atClick Clicks, tag string, value any) {
	ce, ok := value.(CursorEvent)
	if ok && ce.GID == 0 {
		LogWarn("ScheduleAt Gid is 0", "ce", ce)
	}
	se := NewSchedElement(atClick, tag, value)
	theScheduler.savePendingSchedEvent(se)
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
	defer sched.pendingMutex.Unlock()

	for _, se := range sched.pendingScheduled {
		theScheduler.insertScheduleElement(se)
	}
	sched.pendingScheduled = nil
}

// Start runs the scheduler and never returns
func (sched *Scheduler) Start() {

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in Scheduler.Start called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in Scheduler.Start has been called")
			LogError(err, "r", r, "stack", stacktrace)
		}
	}()

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
		theEngine.advanceTransposeTo(newclick)

		SetCurrentClick(newclick)

		sched.handlePendingSchedEvents()

		theProcessManager.checkProcess()
		theAttractManager.checkAttract()
	}
	LogInfo("StartRealtime ends")
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// LogOfType("scheduler", "Scheduler.advanceClickTo", "toClick", toClick, "scheduler", sched)

	// Don't let events get handled while we're advancing
	// XXX - this might not be needed if all communication/syncing
	// is done only from the scheduler loop
	theRouter.inputEventMutex.Lock()
	defer func() {
		theRouter.inputEventMutex.Unlock()
	}()

	doAutoCursorUp := true
	sched.lastClick += 1
	for clk := sched.lastClick; clk <= toClick; clk++ {
		sched.triggerItemsScheduledAtOrBefore(clk)
		// sched.advancePendingNoteOffsByOneClick()
		if doAutoCursorUp {
			theCursorManager.CheckAutoCursorUp()
		}
	}
	sched.lastClick = toClick
}

func (sched *Scheduler) DeleteCursorEventsWhoseGIDIs(gid int) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.GID == gid {
			// LogInfo("DeleteEventsWhoseGidIs", "gid", gid, "i", i)
			sched.schedList.Remove(i)
			// keep going, there will be lots of them
		}
	}
}

// XXX - Fade, Filter, and Delete should be combined into one function

func (sched *Scheduler) FadeEventsWithTag(tag string) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		if se.Tag != tag {
			continue
		}
		// LogInfo("DeleteEventsWithTag Removing schedList entry", "tag", tag, "i", i, "se", se)
		ce, isce := se.Value.(CursorEvent)
		// LogInfo("SAW CURSOREVENT", "v", v, "ddu", v.Ddu)
		if isce {
			ce.Pos.Z *= 0.3
			se.Value = ce
			// LogInfo("FadeEvents", "Z", se.Value.(CursorEvent).Pos.Z)
		}
	}
}

func (sched *Scheduler) FilterEventsWithTag(tag string) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		if se.Tag != tag {
			continue
		}
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.Ddu == "up" && rnd.Float32() < 0.5 {
			theCursorManager.DeleteActiveCursor(ce.GID)
		}
		sched.schedList.Remove(i)
	}
}

func (sched *Scheduler) DeleteEventsWithTag(tag string) {

	sched.mutex.Lock()

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		if se.Tag != tag {
			continue
		}
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.Ddu == "up" {
			theCursorManager.DeleteActiveCursor(ce.GID)
		}
		sched.schedList.Remove(i)
	}

	sched.mutex.Unlock()
}

func (sched *Scheduler) CountEventsWithTag(tag string) int {

	sched.mutex.Lock() // should be Rlock?
	defer sched.mutex.Unlock()

	count := 0
	for i := sched.schedList.Front(); i != nil; i = i.Next() {
		se := i.Value.(*SchedElement)
		if se.Tag == tag {
			count++
		}
	}
	return count
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
			LogOfType("scheduler", "triggerItemsScheduledAtOrBefore: NoteOn", "note", v.String())
			v.Synth.SendNoteToMidiOutput(v)

		case *NoteOff:
			LogOfType("scheduler", "triggerItemsScheduledAtOrBefore: NoteOff", "note", v.String())
			v.Synth.SendNoteToMidiOutput(v)

		case midi.Message:
			synthName, err := GetParam("global.midithrusynth")
			if err != nil {
				LogError(err)
				synthName = ""
			}
			synth := Synths[synthName]
			var bt []byte

			switch {
			case v.GetSysEx(&bt):
				LogWarn("triggerItemsScheduledAtOrBefore: should handle sysex?", "msg", v)

			case v.Is(midi.NoteOnMsg):
				// Need to maintain noteDownCount, don't use SendBytesToMidiOutput
				nt := NewNoteOn(synth, v[1], v[2])
				synth.SendNoteToMidiOutput(nt)

			case v.Is(midi.NoteOffMsg):
				// Need to maintain noteDownCount, don't use SendBytesToMidiOutput
				nt := NewNoteOff(synth, v[1], v[2])
				synth.SendNoteToMidiOutput(nt)

			case v.Is(midi.ProgramChangeMsg):
				synth.SendBytesToMidiOutput([]byte{v[0], v[1]})

			case v.Is(midi.PitchBendMsg):
				synth.SendBytesToMidiOutput([]byte{v[0], v[1], v[2]})

			case v.Is(midi.ControlChangeMsg):
				synth.SendBytesToMidiOutput([]byte{v[0], v[1], v[2]})

			default:
				LogWarn("Unable to handle MIDI input", "msg", v)
			}

		case CursorEvent:
			ce := v
			if ce.GID == 0 {
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
			t := fmt.Sprintf("%T", v)
			LogError(fmt.Errorf("triggerItemsScheduledAtOrBefore: unhandled Value"), "type", t)
		}

		// This is where
		sched.schedList.Remove(i)
		// LogInfo("After Removing from schedList", "i", i, "Len", sched.schedList.Len())
	}

	sched.mutex.Unlock()

	for _, ce := range tobeExecuted {
		theCursorManager.ExecuteCursorEvent(ce)
	}

}

func (sched *Scheduler) ToString() string {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	s := "Scheduler{"
	for i := sched.schedList.Front(); i != nil; i = i.Next() {
		se := i.Value.(*SchedElement)
		switch v := se.Value.(type) {
		/*
			case *Phrase:
				phr := v
				s += fmt.Sprintf("(%d,%v)", pe.AtClick, phr)
		*/
		case *NoteOn:
			s += fmt.Sprintf("(%d,%s)", se.GetClick(), v.String())
		case *NoteOff:
			s += fmt.Sprintf("(%d,%s)", se.GetClick(), v.String())
		case CursorEvent:
			s += fmt.Sprintf("(%d,%v)", v.GetClick(), v)
		default:
			s += fmt.Sprintf("(Unknown Type=%T)", v)
		}
	}
	s += "}"
	return s
}

func (sched *Scheduler) PendingToString() string {

	sched.pendingMutex.Lock()
	defer sched.pendingMutex.Unlock()

	s := "pendingScheduled{"
	for _, se := range sched.pendingScheduled {
		switch v := se.Value.(type) {
		/*
			case *Phrase:
				phr := v
				s += fmt.Sprintf("(%d,%v)", pe.AtClick, phr)
		*/
		case *NoteOn:
			s += fmt.Sprintf("(%d,%s)", se.GetClick(), v.String())
		case *NoteOff:
			s += fmt.Sprintf("(%d,%s)", se.GetClick(), v.String())
		case CursorEvent:
			s += fmt.Sprintf("(%d,%v)", v.GetClick(), v)
		default:
			s += fmt.Sprintf("(Unknown Type=%T)", v)
		}
	}
	s += "}"
	return s
}

func (sched *Scheduler) Format(f fmt.State, c rune) {
	s := sched.ToString()
	_, _ = f.Write([]byte(s)) // XXX - can this ever fail?
}

func (sched *Scheduler) insertScheduleElement(se *SchedElement) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	switch v := (se.Value).(type) {
	case *NoteOn:
	case *NoteOff:
	case CursorEvent:
		if v.Ddu != "clear" && v.GID == 0 {
			LogWarn("insertScheduleElement CursorEvent Gid is empty", "v", v)
		}
	}
	schedClick := se.GetClick()
	LogOfType("scheduler", "Scheduler.insertScheduleElement", "value", se.Value, "click", se.GetClick(), "beforelen", sched.schedList.Len())
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
