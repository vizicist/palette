package kit

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

	AutoTransposeOn     bool
	autoTransposeNext   Clicks
	autoTransposeClicks Clicks // time between auto transpose changes
	autoTransposeIndex  int        // current place in transposeValues
	autoTransposeValues []int

	currentPitchOffset  *atomic.Int32

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

		currentPitchOffset:  &atomic.Int32{},
		AutoTransposeOn:     false,
		autoTransposeNext:   0,
		autoTransposeClicks: 8 * OneBeat,
		autoTransposeIndex:  0,
		autoTransposeValues: []int{0, -2, 3, -5},
	}
	InitializeClicksPerSecond(defaultClicksPerSecond)
	s.currentPitchOffset.Store(0) // probably not needed
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

// NOTE: a pointer is used so se.SetClick() can modify the value
func (se *SchedElement) SetClick(click Clicks) {
	se.AAtClick = click
}

func (se SchedElement) GetClick() Clicks {
	return se.AAtClick
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
	defer sched.pendingMutex.Unlock()

	for _, se := range sched.pendingScheduled {
		TheScheduler.insertScheduleElement(se)
	}
	sched.pendingScheduled = nil
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
		sched.advanceTransposeTo(newclick)

		SetCurrentClick(newclick)

		sched.handlePendingSchedEvents()

		TheAttractManager.checkAttract()

		TheHost.EveryTick()
	}
	LogInfo("StartRealtime ends")
}

func (sched *Scheduler) advanceClickTo(toClick Clicks) {

	// LogOfType("scheduler", "Scheduler.advanceClickTo", "toClick", toClick, "scheduler", sched)

	// Don't let events get handled while we're advancing
	// XXX - this might not be needed if all communication/syncing
	// is done only from the scheduler loop
	TheHost.InputEventLock()
	defer func() {
		TheHost.InputEventUnlock()
	}()

	doAutoCursorUp := true
	sched.lastClick += 1
	for clk := sched.lastClick; clk <= toClick; clk++ {
		sched.triggerItemsScheduledAtOrBefore(clk)
		// sched.advancePendingNoteOffsByOneClick()
		if doAutoCursorUp {
			TheCursorManager.CheckAutoCursorUp()
		}
	}
	sched.lastClick = toClick
}

func (sched *Scheduler) SetTranspose(i int) {
	sched.currentPitchOffset.Store(int32(i))
	LogOfType("transpose", "SetTranspose", "i", i)
	// LogInfo("Engine.SetTranspose", "schedule", TheScheduler.ToString())
	// LogInfo("Engine.SetTranspose", "pending", TheScheduler.PendingToString())
}

func (sched *Scheduler) SetAutoTransposeBeats(beats int) {
	sched.autoTransposeNext = Clicks(beats) * OneBeat
	sched.autoTransposeClicks = Clicks(beats) * OneBeat
	LogOfType("transpose", "SetTransposeBeats", "beats", beats)
}

func (sched *Scheduler) advanceTransposeTo(newclick Clicks) {

	if !sched.AutoTransposeOn {
		return
	}
	if newclick < sched.autoTransposeNext {
		return
	}
	sched.autoTransposeNext = newclick + sched.autoTransposeClicks
	sched.autoTransposeIndex = (sched.autoTransposeIndex + 1) % len(sched.autoTransposeValues)
	sched.SetTranspose(sched.autoTransposeValues[sched.autoTransposeIndex])
	// TheScheduler.SendAllPendingNoteoffs()
}

/*
	XXX - pitchOffset should be done elsewhere
	pitchOffset := TheMidiIO.engineTranspose

	// Hardcoded, channel 10 is usually drums, doesn't get transposed
	// Should probably be an attribute of the Synth.
	const drumChannel = 10
	if TheMidiIO.autoTransposeOn && synth.portchannel.channel != drumChannel {
		pitchOffset += TheMidiIO.autoTransposeValues[TheMidiIO.autoTransposeIndex]
	}
	newpitch := int(pitch) + pitchOffset
	if newpitch < 0 {
		newpitch = 0
	} else if newpitch > 127 {
		newpitch = 127
	}
	if newpitch != int(pitch) {
		LogOfType("midi", "SendNoteToMidiOutput adjusting pitch", "newpitch", newpitch, "oldpitch", pitch)
	}
	pitch = uint8(newpitch)
*/

func (sched *Scheduler) DeleteCursorEventsWhoseGidIs(gid int) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	var nexti *list.Element
	for i := sched.schedList.Front(); i != nil; i = nexti {
		nexti = i.Next()
		se := i.Value.(*SchedElement)
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.Gid == gid {
			// LogInfo("DeleteEventsWhoseGidIs", "gid", gid, "i", i)
			sched.schedList.Remove(i)
			// keep going, there will be lots of them
		}
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
		// LogInfo("DeleteEventsWithTag Removing schedList entry", "tag", tag, "i", i, "se", se)
		ce, isce := se.Value.(CursorEvent)
		// LogInfo("SAW CURSOREVENT", "v", v, "ddu", v.Ddu)
		if isce && ce.Ddu == "up" {
			// LogInfo("UP CURSOREVENT should be removing gid", "gid", v.Gid)
			TheCursorManager.DeleteActiveCursor(ce.Gid)
			// LogInfo("UP CURSOREVENT after removing gid", "gid", v.Gid)
		}
		sched.schedList.Remove(i)
		// keep going, there will be lots of them
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
			LogOfType("scheduler", "triggerItemsScheduleAt: NoteOn", "note", v.String())
			v.Synth.SendNoteToMidiOutput(v)

		case *NoteOff:
			LogOfType("scheduler", "triggerItemsScheduleAt: NoteOff", "note", v.String())
			v.Synth.SendNoteToMidiOutput(v)

		case midi.Message:
			synthName, err := TheHost.GetParam("engine.midithrusynth")
			if err != nil || synthName == "" {
				LogError(err)
				synthName = "default"
			}
			synth := Synths[synthName]
			var bt []byte

			switch {
			case v.GetSysEx(&bt):
				LogWarn("triggerItemsScheduleAtOrBefore: should handle sysex?", "msg", v)

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
