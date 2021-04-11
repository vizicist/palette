package engine

// MIDIDeviceEvent is a single MIDI event
type MIDIDeviceEvent struct {
	Timestamp int64 // milliseconds
	Status    int64
	Data1     int64
	Data2     int64
}

// CursorDeviceEvent is a single CursorDevice event
type CursorDeviceEvent struct {
	NUID      string
	Region    string
	CID       string
	Timestamp int64  // milliseconds
	Ddu       string // "down", "drag", "up"
	X         float32
	Y         float32
	Z         float32
	Area      float32
}

// CursorStepEvent is a down, drag, or up event inside a loop step
type CursorStepEvent struct {
	ID        string // globally unique of the form {nuid}.{CID}[.#{instancenum}]
	X         float32
	Y         float32
	Z         float32
	Ddu       string
	LoopsLeft int
	Fresh     bool
	Quantized bool
	Finished  bool
}

// ActiveStepCursor is a currently active (i.e. down) cursor
// NOTE: these are the cursors caused by CursorStepEvents,
// not the cursors caused by CursorDeviceEvents.
type ActiveStepCursor struct {
	id        string
	x         float32
	y         float32
	z         float32
	loopsLeft int
	maxz      float32
	lastDrag  Clicks // to filter MIDI events for drag
	downEvent CursorStepEvent
}

// CursorDeviceCallbackFunc xxx
type CursorDeviceCallbackFunc func(e CursorDeviceEvent)

// StartCursorInput xxx
func StartCursorInput() {
	go RealStartCursorInput(TheRouter().handleCursorDeviceInput)
}
