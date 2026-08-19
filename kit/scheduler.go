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

const maxSameClickSchedulePasses = 64

type Event any

type Scheduler struct {
	mutex            sync.RWMutex
	schedList        *list.List // of *SchedElements
	lastClick        Clicks
	pendingMutex     sync.RWMutex
	pendingScheduled []*SchedElement

	// State of the realtime loop, carried across ticks. Only tickOnce and
	// Start touch these, both on the scheduler goroutine.
	nonRealtime          bool
	lastMIDINoteWatchdog time.Time
	tickPanics           int
	lastTickPanicLogged  time.Time
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

func (sched *Scheduler) handlePendingSchedEvents() int {
	sched.pendingMutex.Lock()
	pending := sched.pendingScheduled
	sched.pendingScheduled = nil
	sched.pendingMutex.Unlock()

	for _, se := range pending {
		sched.insertScheduleElement(se)
	}
	return len(pending)
}

// triggerClickAndDrain executes everything due at this click, including events
// scheduled by handlers that run during the same click. Keeping this work on
// the scheduler goroutine preserves event serialization while avoiding an
// additional click of latency for CursorEvent -> NoteOn chains.
func (sched *Scheduler) triggerClickAndDrain(click Clicks) {
	sched.handlePendingSchedEvents()
	for pass := 0; pass < maxSameClickSchedulePasses; pass++ {
		sched.triggerItemsScheduledAtOrBefore(click)
		if sched.handlePendingSchedEvents() == 0 {
			return
		}
	}
	LogWarn("Scheduler stopped draining same-click events",
		"click", click,
		"maxpasses", maxSameClickSchedulePasses)
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

	sched.nonRealtime = false
	sched.lastMIDINoteWatchdog = time.Now()

	// By reading from tick.C, we wake up every 2 milliseconds
	for range tick.C {
		sched.tickOnce()
	}
	LogInfo("StartRealtime ends")
}

// tickOnce is one pass of the realtime loop, and it recovers on its own.
//
// The recover belongs here rather than at Start's scope. There are reachable
// panics on this path - a nil Synth from the MIDI-thru lookup, an empty quad
// directory reaching loadQuadRand from attract - and from out there any one of
// them unwinds the whole loop: Start logs it and returns, nothing restarts the
// goroutine, and the engine goes on answering HTTP with its clock stopped. No
// scheduled notes, no cursor handling, no attract, and no hung-note watchdog,
// which lives in this loop too. Nothing outside sees a failure, because the
// process is still up. Recovering per tick costs one dropped 2ms tick instead.
func (sched *Scheduler) tickOnce() {

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		sched.tickPanics++
		// A fault that recurs every tick would write a stack trace every 2ms
		// and fill the disk. Log the first, then at most one a minute, carrying
		// the running count so a persistent one stays visible.
		if !sched.lastTickPanicLogged.IsZero() && time.Since(sched.lastTickPanicLogged) < time.Minute {
			return
		}
		sched.lastTickPanicLogged = time.Now()
		stacktrace := string(debug.Stack())
		fmt.Printf("PANIC: recover in Scheduler tick called, r=%+v stack=%v", r, stacktrace)
		LogError(fmt.Errorf("PANIC: recover in Scheduler tick has been called"),
			"r", r, "panics", sched.tickPanics, "stack", stacktrace)
	}()

	// sched.now = now
	now := time.Now()
	uptimesecs := Uptime()

	// XXX - should lock from here?

	thisClick := CurrentClick()
	var newclick Clicks
	if sched.nonRealtime {
		newclick = thisClick + 1
	} else {
		newclick = Seconds2Clicks(uptimesecs)
	}
	SetCurrentMilli(int64(uptimesecs * 1000.0))

	if newclick <= thisClick {
		// Info("SCHEDULER skipping to next loop, newclick is unchanged","newclick",newclick,"currentClick",currentClick)
		return
	}

	sched.advanceClickTo(newclick)
	theEngine.advanceTransposeTo(newclick)

	theProcessManager.checkProcess()
	theAttractManager.checkAttract()
	if now.Sub(sched.lastMIDINoteWatchdog) >= midiNoteWatchdogInterval {
		SendExpiredMIDINoteOffs(now, maxMIDINoteDuration)
		sched.lastMIDINoteWatchdog = now
	}
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
		// Handlers triggered at this click must observe the click currently
		// being processed, especially when catching up across several clicks.
		SetCurrentClick(clk)
		sched.triggerClickAndDrain(clk)
		// sched.advancePendingNoteOffsByOneClick()
		if doAutoCursorUp {
			theCursorManager.CheckAutoCursorUp()
		}
		if theStepper != nil {
			theStepper.AdvanceTo(clk)
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

func (sched *Scheduler) DeleteSamplePlaybackStarts(tag string, sigilChannel int) {
	deleted := sched.deleteEvents(func(se *SchedElement) bool {
		if se.Tag == tag && samplePlaybackEventChannel(se.Value) == sigilChannel {
			return true
		}
		return false
	})
	if deleted > 0 {
		LogOfType("sampleplayback", "DeleteSamplePlaybackStarts", "tag", tag, "sigilChannel", sigilChannel, "deleted", deleted)
	}
}

func (sched *Scheduler) DeleteSoundEventsWithTag(tag string) int {
	if sched == nil {
		return 0
	}

	deleted := sched.deleteEvents(func(se *SchedElement) bool {
		return se.Tag == tag && isSoundEvent(se.Value)
	})
	if deleted > 0 {
		LogInfo("DeleteSoundEventsWithTag", "tag", tag, "deleted", deleted)
	}
	return deleted
}

func (sched *Scheduler) deleteEvents(match func(*SchedElement) bool) int {
	return sched.deletePendingEvents(match) + sched.deleteScheduledEvents(match, nil)
}

func (sched *Scheduler) deletePendingEvents(match func(*SchedElement) bool) int {
	sched.pendingMutex.Lock()
	defer sched.pendingMutex.Unlock()

	keptPending := sched.pendingScheduled[:0]
	deleted := 0
	for _, se := range sched.pendingScheduled {
		if match(se) {
			deleted++
			continue
		}
		keptPending = append(keptPending, se)
	}
	sched.pendingScheduled = keptPending
	return deleted
}

func (sched *Scheduler) deleteScheduledEvents(match func(*SchedElement) bool, onDelete func(*SchedElement)) int {

	// Remove and collect under the lock, then run the callbacks with it
	// released.
	//
	// onDelete used to be called while sched.mutex was held, and it re-enters
	// the scheduler: DeleteEventsWithTag and FilterEventsWithTag both delete
	// active cursors from it, and DeleteActiveCursor -> cleanupDeletedActiveCursor
	// -> stopActiveSamplePlayback -> stopSamplePlayback calls
	// DeleteSamplePlaybackStarts, which locks this same mutex (and then
	// ScheduleAt, which wants the pending list). sync.RWMutex is not reentrant,
	// so a sample-backed cursor being cleaned up here wedged the goroutine
	// outright - and one of those callers runs on the scheduler goroutine.
	// clearActiveCursors in cursor.go already avoids this the same way.
	var deleted []*SchedElement
	func() {
		sched.mutex.Lock()
		defer sched.mutex.Unlock()

		var nexti *list.Element
		for i := sched.schedList.Front(); i != nil; i = nexti {
			nexti = i.Next()
			se := i.Value.(*SchedElement)
			if !match(se) {
				continue
			}
			sched.schedList.Remove(i)
			deleted = append(deleted, se)
		}
	}()

	if onDelete != nil {
		for _, se := range deleted {
			onDelete(se)
		}
	}
	return len(deleted)
}

func isSoundEvent(value any) bool {
	switch value.(type) {
	case *NoteOn, *NoteOff, *PitchBend, *StepperSamplePlaybackStop,
		*SamplePlaybackStart, *SamplePlaybackStop, *SamplePlaybackPitch:
		return true
	default:
		return false
	}
}

func samplePlaybackEventChannel(value any) int {
	switch v := value.(type) {
	case *SamplePlaybackStart:
		return v.SigilChannel
	case *SamplePlaybackPitch:
		return v.SigilChannel
	default:
		return -1
	}
}

// forEachEventWithTag applies fn to every scheduled element with the given
// tag, in place (no removal). Compare deleteScheduledEvents, which removes.
// forEachPendingEventWithTag applies fn to the not-yet-scheduled events of one
// tag. forEachEventWithTag only walks schedList, so without this the pending
// events of a patch were never faded, only thrown away.
func (sched *Scheduler) forEachPendingEventWithTag(tag string, fn func(*SchedElement)) {
	sched.pendingMutex.Lock()
	defer sched.pendingMutex.Unlock()

	for _, se := range sched.pendingScheduled {
		if se.Tag == tag {
			fn(se)
		}
	}
}

func (sched *Scheduler) forEachEventWithTag(tag string, fn func(*SchedElement)) {
	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	for i := sched.schedList.Front(); i != nil; i = i.Next() {
		se := i.Value.(*SchedElement)
		if se.Tag == tag {
			fn(se)
		}
	}
}

func (sched *Scheduler) FadeEventsWithTag(tag string) {
	fade := func(se *SchedElement) {
		if ce, isce := se.Value.(CursorEvent); isce {
			ce.Pos.Z *= 0.3
			se.Value = ce
		}
	}
	sched.forEachEventWithTag(tag, fade)
	// Pending events belong to this patch too, and fading them is the point -
	// patch.go used to deal with them by discarding the entire pending list.
	sched.forEachPendingEventWithTag(tag, fade)
}

func (sched *Scheduler) FilterEventsWithTag(tag string) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	sched.deleteScheduledEvents(func(se *SchedElement) bool {
		return se.Tag == tag
	}, func(se *SchedElement) {
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.Ddu == "up" && rnd.Float32() < 0.5 {
			theCursorManager.DeleteActiveCursor(ce.GID)
		}
	})
	sched.deletePendingEvents(func(se *SchedElement) bool {
		return se.Tag == tag
	})
}

func (sched *Scheduler) DeleteEventsWithTag(tag string) {
	sched.deleteScheduledEvents(func(se *SchedElement) bool {
		return se.Tag == tag
	}, func(se *SchedElement) {
		ce, isce := se.Value.(CursorEvent)
		if isce && ce.Ddu == "up" {
			theCursorManager.DeleteActiveCursor(ce.GID)
		}
	})
	sched.deletePendingEvents(func(se *SchedElement) bool {
		return se.Tag == tag
	})
}

func (sched *Scheduler) CountEventsWithTag(tag string) int {

	sched.mutex.RLock()
	defer sched.mutex.RUnlock()

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

		case *PitchBend:
			LogOfType("scheduler", "triggerItemsScheduledAtOrBefore: PitchBend", "pitchbend", v.String())
			if v.Synth != nil {
				v.Synth.SendPitchBend(v.Value)
			}

		case *StepperSamplePlaybackStop:
			if theStepper != nil {
				noteOff := theStepper.SamplePlaybackStopIfCurrent(v)
				if noteOff != nil {
					LogOfType("scheduler", "triggerItemsScheduledAtOrBefore: StepperSamplePlaybackStop", "note", noteOff.String())
					noteOff.Synth.SendNoteToMidiOutput(noteOff)
				}
			}

		case *SamplePlaybackStart:
			LogOfType("scheduler", "triggerItemsScheduledAtOrBefore: SamplePlaybackStart", "patch", v.Patch, "sigilChannel", v.SigilChannel, "sampleSelector", v.SampleSelector)
			v.Trigger()

		case *SamplePlaybackStop:
			LogOfType("scheduler", "triggerItemsScheduledAtOrBefore: SamplePlaybackStop", "patch", v.Patch, "sigilChannel", v.SigilChannel, "sampleSelector", v.SampleSelector)
			v.Trigger()

		case *SamplePlaybackPitch:
			LogOfType("scheduler", "triggerItemsScheduledAtOrBefore: SamplePlaybackPitch", "patch", v.Patch, "sigilChannel", v.SigilChannel, "value", v.Value)
			v.Trigger()

		case midi.Message:
			synthName, err := GetParam("global.midithrusynth")
			if err != nil {
				LogError(err)
				synthName = ""
			}
			// GetSynth rather than a raw map lookup. global.midithrusynth is a
			// free-form string that nothing validates on set, so a typo or a
			// name missing from Synths.json gives a nil *Synth, and every
			// branch below dereferences it - SendNoteToMidiOutput reaches
			// synth.state through midiOutputEnabled. GetSynth warns and falls
			// back to the dummy synth registered under "".
			synth := GetSynth(synthName)
			if synth == nil {
				LogWarn("MIDI thru: no synth to send to", "synth", synthName)
				break
			}
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

	sched.mutex.RLock()
	defer sched.mutex.RUnlock()

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

	sched.pendingMutex.RLock()
	defer sched.pendingMutex.RUnlock()

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
	// Write errors to a fmt.State are handled inside the fmt package;
	// there's nothing useful a Formatter can do with them.
	fmt.Fprint(f, sched.ToString())
}

func (sched *Scheduler) insertScheduleElement(se *SchedElement) {

	sched.mutex.Lock()
	defer sched.mutex.Unlock()

	switch v := (se.Value).(type) {
	case *NoteOn:
	case *NoteOff:
	case *PitchBend:
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
