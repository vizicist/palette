package engine

import (
	"encoding/json"
	"strings"

	"github.com/hypebeast/go-osc/osc"
	midi "gitlab.com/gomidi/midi/v2"
)

type RecordingEvent struct {
	Event       string
	Click       Clicks
	CursorEvent *CursorEvent
	MidiEvent   *MidiEvent
	OscEvent    *OscEvent
}

type MIDIEventHandler func(MidiEvent)

// CursorEvent is a singl Cursor event
type CursorEvent struct {
	Cid   string `json:"Cid"`
	Click Clicks `json:"Click"`
	// Source string
	Ddu  string  `json:"Ddu"` // "down", "drag", "up" (sometimes "clear")
	X    float32 `json:"X"`
	Y    float32 `json:"Y"`
	Z    float32 `json:"Z"`
	Area float32 `json:"Area"`
}

// OscEvent is an OSC message
type OscEvent struct {
	Msg    *osc.Message
	Source string
}

type MidiEvent struct {
	Msg midi.Message
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

func (ce CursorEvent) String() string {
	bytes, err := json.Marshal(ce)
	if err != nil {
		return "{\"error\":\"Unable to Marshal CursorEvent\"}"
	}
	return string(bytes)
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

func (event OscEvent) String() string {
	return "{\"event\":\"osc\",\"source\":\"" + event.Source + "\",\"msg\":\"" + event.Msg.String() + "\"}"
}

func (event MidiEvent) String() string {
	return "{\"event\":\"midi\",\"msg\":\"" + event.Msg.String() + "\"}"
}
