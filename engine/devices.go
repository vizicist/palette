package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// MIDIDeviceEvent is a single MIDI event
type MIDIDeviceEvent struct {
	Timestamp int64 // milliseconds
	Status    int64
	Data1     int64
	Data2     int64
}

// CursorDeviceEvent is a single CursorDevice event
type CursorDeviceEvent struct {
	ID        string
	Source    string
	Timestamp time.Time
	Ddu       string // "down", "drag", "up" (sometimes "clear")
	X         float32
	Y         float32
	Z         float32
	Area      float32
}

/*
// CursorStepEvent is a down, drag, or up event inside a loop step
type CursorStepEvent struct {
	ID        string // globally unique of the form {Source}[.#{instancenum}]
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
	// id        string
	x float32
	y float32
	z float32
	// loopsLeft int
	maxzSoFar float32
	lastDrag  Clicks // to filter MIDI events for drag
	downEvent CursorStepEvent
}
*/

// CursorDeviceCallbackFunc xxx
type CursorDeviceCallbackFunc func(e CursorDeviceEvent, lockit bool)

// MorphDefs xxx
var MorphDefs map[string]string

// LoadMorphs initializes the list of morphs
func LoadMorphs() error {

	MorphDefs = make(map[string]string)

	// If you have more than one morph, or
	// want the player assignment to NOT be
	// automatice, put them in here.
	path := ConfigFilePath("morphs.json")
	if !fileExists(path) {
		return fmt.Errorf("unable to get path to morphs.json")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil // It's okay if file isn't present
	}
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s, err=%s", path, err)
	}
	toplevel := f.(map[string]interface{})

	for serialnum, playerinfo := range toplevel {
		playername := playerinfo.(string)
		if Debug.Morph {
			log.Printf("Setting Morph serial=%s player=%s\n", serialnum, playername)
		}
		MorphDefs[serialnum] = playername
		// TheRouter().setPlayerForMorph(serialnum, playername)
	}
	return nil
}
