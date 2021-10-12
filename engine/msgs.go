package engine

import (
	"fmt"
	"log"

	"github.com/vizicist/portmidi"
)

//////////////////////////////////////////////////////////////////

func (note *Note) BytesString() string {
	return "BytesStringNeedsImplementing"
}

func NewNoteMsg(note *Note) string {
	return fmt.Sprintf(
		"\"type\":\"%s\","+
			"\"clicks\":\"%d\","+
			"\"duration\":\"%d\","+
			"\"pitch\":\"%d\","+
			"\"velocity\":\"%d\","+
			"\"sound\":\"%s\","+
			"\"bytes\":\"%s\"",
		note.TypeOf,
		note.Clicks,
		note.Duration,
		note.Pitch,
		note.Velocity,
		note.Sound,
		note.BytesString())
}

// MIDIDeviceMsg is a single MIDI event
type MIDIDeviceMsg struct {
	Timestamp int64 // milliseconds
	Status    int64
	Data1     int64
	Data2     int64
}

func NewMIDIDeviceMsg(e portmidi.Event) string {
	return fmt.Sprintf(
		`"timestamp":"%d","status":"%d","data1":"%d","data2":"%d"`,
		e.Timestamp, e.Status, e.Data1, e.Data2)
}

// Cursor3DDeviceMsg is a single Cursor3DDevice event
type Cursor3DDeviceMsg struct {
	Region     string
	ID         string
	Timestamp  int64  // milliseconds, absolute
	DownDragUp string // "down", "drag", "up"
	X          float32
	Y          float32
	Z          float32
	Area       float32
}

// Cursor3DMsg is a down, drag, or up event inside a loop step
type Cursor3DMsg struct {
	ID         string // globally unique?  All Msgs for a given Cursor3D have the same ID
	X          float32
	Y          float32
	Z          float32
	DownDragUp string
	LoopsLeft  int
	Fresh      bool
	Quantized  bool
	Finished   bool
}

// ActiveStepCursor3D is a currently active (i.e. down) cursor
// NOTE: these are the cursors caused by Cursor3DMsgs,
// not the cursors caused by Cursor3DDeviceMsgs.
type ActiveStepCursor3D struct {
	id        string
	x         float32
	y         float32
	z         float32
	loopsLeft int
	maxz      float32
	lastDrag  Clicks // to filter MIDI events for drag
	downMsg   Cursor3DMsg
}

// Cursor3DDeviceCallbackFunc xxx
type Cursor3DDeviceCallbackFunc func(e Cursor3DDeviceMsg)

func NewSimpleCmd(subj string) Cmd {
	return Cmd{Subj: subj, Values: nil}
}

func NewNoteCmd(note *Note) Cmd {
	values := make(map[string]string)
	values["note"] = note.ToString()
	return Cmd{Subj: "note", Values: values}
}

func NewSetParamMsg(name string, value string) string {
	return fmt.Sprintf("\"name\":\"%s\",\"value\":\"%s\"", name, value)
}

////////////////////////////////////////////////////////////////////

// NewClickMsg xxx
func NewClickMsg(click Clicks) string {
	return fmt.Sprintf("\"click\":\"%d\"", click)
}

//////////////////////////////////////////////////////////////////

// ToBool xxx
func ToBool(arg interface{}) bool {
	r, ok := arg.(bool)
	if !ok {
		log.Printf("Unable to convert interface to bool!\n")
		r = false
	}
	return r
}

// ToString xxx
func ToString(arg interface{}) string {
	if arg == nil {
		return ""
	}
	r, ok := arg.(string)
	if !ok {
		log.Printf("Unable to convert interface to string!\n")
		r = ""
	}
	return r
}
