package engine

// CursorDown etc match values in sensel.h
const (
	CursorDown = 1
	CursorDrag = 2
	CursorUp   = 3
)

// CursorDevice xxx
type CursorDevice interface {
}

// CursorDeviceEvent is a single CursorDevice event
type CursorDeviceEvent struct {
	Cid        string // of the form {source}.{cursorname}
	Timestamp  int64  // milliseconds
	DownDragUp string // "down", "drag", "up"
	X          float32
	Y          float32
	Z          float32
	Area       float32
}

// CursorStepEvent is a down, drag, or up event inside a loop step
type CursorStepEvent struct {
	// Source     string
	// Region     string
	ID         string // globally unique of the form {source}.{region}.{cursorID}
	X          float32
	Y          float32
	Z          float32
	Downdragup string
	LoopsLeft  int
	Fresh      bool
	Quantized  bool
	Finished   bool
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
	go RealStartCursorInput(TheRouter().HandleDeviceCursorInput)
}
