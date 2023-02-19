package engine

import (
	"encoding/json"
	"strings"

	"github.com/hypebeast/go-osc/osc"
	midi "gitlab.com/gomidi/midi/v2"
)

type RecordingEvent struct {
	Event       string
	Value       any
	// PlaybackEvent *PlaybackEvent `json:"PlaybackEvent"`
	// CursorEvent   *CursorEvent   `json:"CursorEvent"`
	// MidiEvent     *MidiEvent     `json:"MidiEvent"`
	// OscEvent      *OscEvent      `json:"OscEvent"`
}

// PlaybackEvent is for start/stop
type PlaybackEvent struct {
	Click     Clicks `json:"Click"`
	IsRunning bool   `json:"IsRunning"`
}

type MIDIEventHandler func(MidiEvent)

// CursorEvent is a singl Cursor event
type CursorEvent struct {
	Click Clicks `json:"Click"`
	Cid   string `json:"Cid"`
	// Source string
	Ddu  string  `json:"Ddu"` // "down", "drag", "up" (sometimes "clear")
	X    float32 `json:"X"`
	Y    float32 `json:"Y"`
	Z    float32 `json:"Z"`
	Area float32 `json:"Area"`
}

// OscEvent is an OSC message
type OscEvent struct {
	Click  Clicks       `json:"Click"`
	Msg    *osc.Message `json:"Msg"`
	Source string       `json:"Source"`
}

type MidiEvent struct {
	Click Clicks
	Msg   midi.Message
}

////////////////////////// CursorEvent methods

func (ce CursorEvent) IsInternal() bool {
	return strings.Contains(ce.Cid, "internal")
}

func (ce CursorEvent) Source() string {
	if ce.Cid == "" {
		LogWarn("CursorEvent.Source: empty cid", "ce", ce)
		return "dummysource"
	}
	arr := strings.Split(ce.Cid, "#")
	return arr[0]
}

/*
func (ce CursorEvent) String() string {
	bytes, err := json.Marshal(ce)
	if err != nil {
		return "{\"error\":\"Unable to Marshal CursorEvent\"}"
	}
	return string(bytes)
}
*/

// XXX - can this make use of generics?  (across all the Event types)
func (ce CursorEvent) Marshal() (bytes []byte, err error) {
	bytes, err = json.Marshal(ce)
	return
}

func CursorEventFromString(s string) (ce CursorEvent, err error) {
	err = json.Unmarshal([]byte(s), &ce)
	return
}

////////////////////////// MidiEvent methods

func (me MidiEvent) HasPitch() bool {
	return me.Msg.Is(midi.NoteOnMsg) || me.Msg.Is(midi.NoteOffMsg)
}

func (me MidiEvent) Pitch() uint8 {
	b := me.Msg.Bytes()
	if len(b) == 3 {
		return b[1]
	}
	LogWarn("MidiEvent.Pitch: bad message length")
	return 0
}

func (e MidiEvent) Marshal() (bytes []byte, err error) {
	bytes, err = json.Marshal(e)
	return
}

func (e MidiEvent) String() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "{\"error\":\"Unable to Marshal CursorEvent\"}"
	}
	return string(bytes)
}

////////////////////////// OscEvent methods

func (e OscEvent) String() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "{\"error\":\"Unable to Marshal CursorEvent\"}"
	}
	return string(bytes)
}

func (e OscEvent) Marshal() (bytes []byte, err error) {
	bytes, err = json.Marshal(e)
	return
}

// func (event MidiEvent) String() string {
// 	return "{\"event\":\"midi\",\"msg\":\"" + event.Msg.String() + "\"}"
// }
