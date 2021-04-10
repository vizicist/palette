package engine

import (
	"log"
	"sync"
)

// LoopEvent is what gets played back in a loop
type LoopEvent struct {
	hasCursor       bool
	cursorStepEvent CursorStepEvent
}

// Step is one step of one loop in the looper
type Step struct {
	events []*LoopEvent
}

// StepLoop is one loop
type StepLoop struct {
	currentStep Clicks
	length      Clicks
	stepsMutex  sync.RWMutex
	steps       []*Step
}

// SetLength changesthe length of a loop
func (loop *StepLoop) SetLength(nclicks Clicks) {

	loop.stepsMutex.Lock()
	defer loop.stepsMutex.Unlock()

	if nclicks < loop.length {
		loop.steps = loop.steps[:nclicks]
		loop.length = nclicks
	} else if nclicks > loop.length {
		for i := loop.length; i < nclicks; i++ {
			step := new(Step)
			step.events = loop.steps[i%loop.length].events
			loop.steps = append(loop.steps, step)
		}
		loop.length = nclicks
	}
	if loop.currentStep >= loop.length {
		loop.currentStep = loop.currentStep % loop.length
	}
}

// Clear removes everything from Loop
func (loop *StepLoop) Clear() {

	loop.stepsMutex.Lock()
	defer loop.stepsMutex.Unlock()

	for i := range loop.steps {
		loop.steps[i].events = nil
	}
}

// ClearID removes all cursorStepEvents with a given id
func (loop *StepLoop) ClearID(id string) {

	loop.stepsMutex.Lock()
	defer loop.stepsMutex.Unlock()

	if DebugUtil.Cursor {
		log.Printf("StepLoop.ClearID: START ClearID for id=%s\n", id)
	}
	for _, step := range loop.steps {
		// This method of deleting things from an array without
		// allocating a new array is found on this page:
		// https://vbauerster.github.io/2017/04/removing-items-from-a-slice-while-iterating-in-go/
		// log.Printf("Before deleting id=%s  events=%v\n", id, step.events)
		newevents := step.events[:0]
		for _, event := range step.events {
			if event.cursorStepEvent.ID != id {
				newevents = append(newevents, event)
				// log.Printf("Should be deleting cursorStepEvent = %v\n", event.cursorStepEvent)
			}
		}
		step.events = newevents
		// log.Printf("After deleting id=%s  events=%v\n", id, step.events)
	}
}

// AddToStep adds a StepItem to the loop at the current step
func (loop *StepLoop) AddToStep(ce CursorStepEvent, stepnum Clicks) {

	loop.stepsMutex.Lock()
	defer loop.stepsMutex.Unlock()

	if DebugUtil.Cursor || DebugUtil.MIDI {
		log.Printf("StepLoop.AddToStep: stepnum=%d ddu=%s cid=%s\n", stepnum, ce.Downdragup, ce.ID)
	}

	step := loop.steps[stepnum]
	le := &LoopEvent{cursorStepEvent: ce}

	// We only want a single drag or down event per cursor in a single Step.
	// If one (of either type) is found for the same cursor id,
	// replace it rather than appending a second one.
	if ce.Downdragup == "drag" || ce.Downdragup == "down" {
		replace := -1
		for i, e := range step.events {
			if ce.ID != e.cursorStepEvent.ID {
				// It's not the same cursor id, ignore it
				continue
			}
			// If we're adding a drag, and find an existing "down"...
			if ce.Downdragup == "drag" &&
				e.cursorStepEvent.Downdragup == "down" {
				log.Printf("Hey, drag with down in same step!? ce.ID=%s, not replacing\n", ce.ID)
				// don't replace
				break
			}
			// If we're adding a drag, and find an existing "drag"...
			if ce.Downdragup == "drag" &&
				e.cursorStepEvent.Downdragup == "drag" {
				replace = i
				break
			}
			// If we're adding a down, and find an existing "down"...
			if ce.Downdragup == "down" &&
				e.cursorStepEvent.Downdragup == "down" {
				log.Printf("Hey, down with down in same step!? ce.ID=%s\n", ce.ID)
				replace = i
				break
			}
			// If we're adding a down, and find an existing "drag"...
			if ce.Downdragup == "down" &&
				e.cursorStepEvent.Downdragup == "drag" {
				log.Printf("Hey, down with drag in same step!? ce.ID=%s\n", ce.ID)
				replace = i
				break
			}
		}
		if replace >= 0 {
			if DebugUtil.Loop || DebugUtil.MIDI {
				log.Printf("Replacing %s event %d in stepnum=%d\n", ce.Downdragup, replace, stepnum)
			}
			step.events[replace] = le
			return
		}
	}

	step.events = append(step.events, le)
	if DebugUtil.Loop {
		log.Printf("AddToStep stepnum=%d #events=%d currentStep=%d ce=%+v\n", stepnum, len(step.events), loop.currentStep, ce)
	}
}

// NewLoop allocates and adds a new steploop
func NewLoop(nclicks Clicks) *StepLoop {
	loop := new(StepLoop)
	loop.length = nclicks

	loop.stepsMutex.Lock()
	defer loop.stepsMutex.Unlock()

	loop.steps = make([]*Step, nclicks)
	for n := range loop.steps {
		loop.steps[n] = new(Step)
		loop.steps[n].events = nil
	}
	return loop
}

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec int) {
	clicksPerSecond = clkpersec
	currentMilliOffset = 0
	currentClickOffset = 0
	oneBeat = Clicks(clicksPerSecond / 2)
}

// ChangeClicksPerSecond is what you use to change the tempo
func ChangeClicksPerSecond(factor float64) {
	TempoFactor = factor
	clkpersec := int(defaultClicksPerSecond * factor)
	if clkpersec < minClicksPerSecond {
		clkpersec = minClicksPerSecond
	}
	if clkpersec > maxClicksPerSecond {
		clkpersec = maxClicksPerSecond
	}
	currentMilliOffset = CurrentMilli
	currentClickOffset = currentClick
	clicksPerSecond = clkpersec
	oneBeat = Clicks(clicksPerSecond / 2)
}

// Seconds2Clicks converts a Time value (elapsed seconds) to Clicks
func Seconds2Clicks(tm float64) Clicks {
	return currentClickOffset + Clicks(0.5+float64(tm*1000-float64(currentMilliOffset))*(float64(clicksPerSecond)/1000.0))
}
