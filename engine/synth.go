package engine

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/vizicist/portmidi"
)

type Synth struct {
	port     string
	channel  int // 0-15 for MIDI channels 1-16
	midiOut  *MidiOutput
	noteDown []bool
}

var Synths map[string]*Synth

func InitSynths() {

	Synths = make(map[string]*Synth)

	filename := ConfigFilePath("synths.json")
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	type synth struct {
		Name    string `json:"name"`
		Port    string `json:"port"`
		Channel int    `json:"channel"`
	}

	var jsynths struct {
		Synths []synth `json:"synths"`
	}
	json.Unmarshal(bytes, &jsynths)

	synthoutput := ConfigBool("generatesound")

	for i := range jsynths.Synths {
		nm := jsynths.Synths[i].Name
		port := jsynths.Synths[i].Port
		channel := jsynths.Synths[i].Channel
		var midiOut *MidiOutput
		if synthoutput {
			midiOut = MIDI.NewMidiOutput(port, channel)
		} else {
			midiOut = MIDI.NewFakeMidiOutput(port, channel)
		}
		Synths[nm] = NewSynth(port, channel, midiOut)
	}
	log.Printf("Synths loaded, len=%d\n", len(Synths))
}

func NewSynth(port string, channel int, midiOut *MidiOutput) *Synth {
	return &Synth{
		port:     port,
		channel:  channel,
		midiOut:  midiOut,
		noteDown: make([]bool, 128),
	}
}

// SendANO sends all-notes-off
func SendANOToSynth(synthName string) {
	synth, ok := Synths[synthName]
	if !ok {
		log.Printf("SendANOToSynth: no such synth - %s\n", synthName)
		return
	}
	status := 0xb0 | (synth.channel - 1)
	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(status),
		Data1:     int64(0x7b),
		Data2:     int64(0x00),
	}
	for i := range synth.noteDown {
		synth.noteDown[i] = false
	}
	// log.Printf("Sending ANO portmidi.Event = %s\n", e)
	SendEvent(synth.midiOut, []portmidi.Event{e})
}

// SendNote sends MIDI output for a Note
func SendNoteToSynth(note *Note) {
	synth, ok := Synths[note.Sound]
	if !ok {
		log.Printf("SendNoteToSynth: no such synth - %s\n", note.Sound)
		return
	}
	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(synth.channel - 1), // pre-populate with the channel
		Data1:     int64(note.Pitch),
		Data2:     int64(note.Velocity),
	}
	if note.TypeOf == NOTEON && note.Velocity == 0 {
		// log.Printf("MIDIIO.SendNote: NOTEON with velocity==0 is a NOTEOFF\n")
		note.TypeOf = NOTEOFF
	}
	switch note.TypeOf {
	case NOTEON:
		e.Status |= 0x90
		if synth.noteDown[note.Pitch] {
			if DebugUtil.MIDI {
				log.Printf("SendNoteToSynth: Ignoring second NOTEON for chan=%d pitch=%d\n", synth.channel, note.Pitch)
			}
			return
		}
		synth.noteDown[note.Pitch] = true
	case NOTEOFF:
		e.Status |= 0x80
		synth.noteDown[note.Pitch] = false
	case CONTROLLER:
		e.Status |= 0xB0
	case PROGCHANGE:
		e.Status |= 0xC0
	case CHANPRESSURE:
		e.Status |= 0xD0
	case PITCHBEND:
		e.Status |= 0xE0
	default:
		log.Printf("SendNoteToSynth: can't handle Note TypeOf=%v\n", note.TypeOf)
		return
	}

	// log.Printf("Sending portmidi.Event = %s\n", e)
	SendEvent(synth.midiOut, []portmidi.Event{e})
}
