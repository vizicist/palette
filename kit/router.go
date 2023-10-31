package kit

import (
	"runtime/debug"
	"sync"
	"time"
)

var OscInputChan  chan OscEvent
var MidiInputChan chan MidiEvent
var CursorInput   chan CursorEvent
var KillMe bool

var InputEventMutex sync.RWMutex

func InputEventLock() {
	InputEventMutex.Lock()
}

func InputEventUnlock() {
	InputEventMutex.Unlock()
}

// InputListener listens for local device inputs (OSC, MIDI)
// them in a single select eliminates some need for locking.
func InputListener() {

	for ! KillMe {
		InputListenOnce()
	}
	LogInfo("InputListener is being killed")
}

func InputListenOnce() {
	defer func() {
		if err := recover(); err != nil {
			LogWarn("Panic in InputListener", "err", err, "stacktrace", string(debug.Stack()))
		}
	}()
	select {
	case msg := <-OscInputChan:
		handleOscInput(msg)
		// TheKit.RecordOscEvent(&msg)
	case event := <-MidiInputChan:
		handleMidiEvent(event)
		// TheEngine.RecordMidiEvent(&event)
	case event := <-CursorInput:
		ScheduleAt(CurrentClick(), event.Tag, event)
	default:
		time.Sleep(time.Millisecond)
	}
}