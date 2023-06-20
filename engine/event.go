package engine

import (
	"encoding/json"
	"strings"

	"github.com/hypebeast/go-osc/osc"
	midi "gitlab.com/gomidi/midi/v2"
)

type RecordingEvent struct {
	Event string
	Value any
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

type OscEvent struct {
	Click  Clicks       `json:"Click"`
	Msg    *osc.Message `json:"Msg"`
	Source string       `json:"Source"`
}

type MidiEvent struct {
	Click Clicks
	Tag   string
	Msg   midi.Message
}

func NewMidiEvent(click Clicks, tag string, msg midi.Message) MidiEvent {
	return MidiEvent{
		Click: click,
		Tag:   tag,
		Msg:   msg,
	}
}

////////////////////////// CursorEvent methods

func (ce CursorEvent) IsInternal() bool {
	return strings.Contains(ce.Tag, "internal")
}

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
