package engine

// MIDIDeviceEvent is a single MIDI event
type MIDIDeviceEvent struct {
	Timestamp int64 // milliseconds
	Status    int64
	Data1     int64
	Data2     int64
}

// GestureDeviceEvent is a single GestureDevice event
type GestureDeviceEvent struct {
	NUID       string
	Region     string
	ID         string
	Timestamp  int64  // milliseconds
	DownDragUp string // "down", "drag", "up"
	X          float32
	Y          float32
	Z          float32
	Area       float32
}

// GestureStepEvent is a down, drag, or up event inside a loop step
type GestureStepEvent struct {
	ID         string // globally unique of the form {nuid}.{CID}[.#{instancenum}]
	X          float32
	Y          float32
	Z          float32
	Downdragup string
	LoopsLeft  int
	Fresh      bool
	Quantized  bool
	Finished   bool
}

// ActiveStepGesture is a currently active (i.e. down) cursor
// NOTE: these are the cursors caused by GestureStepEvents,
// not the cursors caused by GestureDeviceEvents.
type ActiveStepGesture struct {
	id        string
	x         float32
	y         float32
	z         float32
	loopsLeft int
	maxz      float32
	lastDrag  Clicks // to filter MIDI events for drag
	downEvent GestureStepEvent
}

// GestureDeviceCallbackFunc xxx
type GestureDeviceCallbackFunc func(e GestureDeviceEvent)

// StartGestureInput xxx
func StartGestureInput() {
	go RealStartGestureInput(TheRouter().handleGestureDeviceInput)
}
