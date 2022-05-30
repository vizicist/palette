package engine

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/vizicist/portmidi"
)

type Synth struct {
	port        string
	channel     int // 0-15 for MIDI channels 1-16
	bankprogram BankProgram
	midiOut     *MidiOutput
	noteDown    []bool
}

type BankProgram struct {
	bank    int
	program int
}

// Keep track of which BankProgram is currently used on a given MidiOutput,
// se we only send bank/program things when it changes
var MidiOutBankProgram map[*MidiOutput]BankProgram

var Synths map[string]*Synth

func InitSynths() {

	Synths = make(map[string]*Synth)
	MidiOutBankProgram = make(map[*MidiOutput]BankProgram)

	filename := ConfigFilePath("synths.json")
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("InitSynths: ReadFile of %s failed, err=%s\n", filename, err)
		return
	}

	type synth struct {
		Name    string `json:"name"`
		Port    string `json:"port"`
		Channel int    `json:"channel"`
		Bank    int    `json:"bank"`
		Program int    `json:"program"`
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
		bank := jsynths.Synths[i].Bank
		program := jsynths.Synths[i].Program
		var midiOut *MidiOutput
		if synthoutput {
			midiOut = MIDI.openOutput(port)
		} else {
			midiOut = MIDI.openFakeOutput(port)
		}
		if midiOut == nil {
			log.Printf("InitSynths: Unable to open midi port=%s\n", port)
			Synths[nm] = nil
		} else {
			Synths[nm] = NewSynth(port, channel, bank, program, midiOut)
		}
	}
	log.Printf("Synths loaded, len=%d\n", len(Synths))
}

func NewSynth(port string, channel int, bank int, program int, midiOut *MidiOutput) *Synth {
	return &Synth{
		port:        port,
		channel:     channel,
		bankprogram: BankProgram{bank, program},
		midiOut:     midiOut,
		noteDown:    make([]bool, 128),
	}
}

// SendANO sends all-notes-off
func SendANOToSynth(synthName string) {
	synth, ok := Synths[synthName]
	if !ok {
		log.Printf("SendANOToSynth: no such synth - %s\n", synthName)
		return
	}
	if synth == nil {
		// We don't complain, we assume the inability to open the
		// synth named synthName has already been logged.
		return
	}
	status := 0xb0 | (synth.channel - 1)
	for i := range synth.noteDown {
		synth.noteDown[i] = false
	}
	// log.Printf("SendANOToSynth: synth=%s\n", synthName)
	synth.midiOut.stream.WriteShort(int64(status), int64(0x7b), int64(0x00))
}

func SendControllerToSynth(sound string, cnum int, cval int) {
	synth, ok := Synths[sound]
	if !ok {
		log.Printf("SendNoteToSynth: no such synth - %s\n", sound)
		return
	}
	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(synth.channel - 1),
		Data1:     int64(cnum),
		Data2:     int64(cval),
	}
	e.Status |= 0xb0
	if Debug.MIDI {
		log.Printf("SendControllerToSynth: synth=%s status=0x%02x data1=%d data2=%d\n", synth.midiOut.Name(), e.Status, e.Data1, e.Data2)
	}
	if e.Data2 > 0x7f {
		log.Printf("SendControllerToSynth: Hey! Data2 shouldn't be > 0x7f\n")
	} else {
		synth.midiOut.stream.WriteShort(e.Status, e.Data1, e.Data2)
	}
}

// SendNote sends MIDI output for a Note
func SendNoteToSynth(note *Note) {
	sound := note.Sound
	synth, ok := Synths[sound]
	if !ok {
		log.Printf("SendNoteToSynth: no such synth - %s\n", sound)
		return
	}
	if synth == nil {
		// We don't complain, we assume the inability to open the
		// synth named synthName has already been logged.
		return
	}
	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(synth.channel - 1),
		Data1:     int64(note.Pitch),
		Data2:     int64(note.Velocity),
	}
	if note.TypeOf == "noteon" && note.Velocity == 0 {
		log.Printf("MIDIIO.SendNote: noteon with velocity==0 is changed to a noteoff\n")
		note.TypeOf = "noteoff"
	}
	switch note.TypeOf {
	case "noteon":
		e.Status |= 0x90
		if synth.noteDown[note.Pitch] {
			if Debug.MIDI {
				log.Printf("SendNoteToSynth: Ignoring second noteon for chan=%d pitch=%d\n", synth.channel, note.Pitch)
			}
			return
		}
		synth.noteDown[note.Pitch] = true
	case "noteoff":
		e.Status |= 0x80
		e.Data2 = 0
		synth.noteDown[note.Pitch] = false
	case "controller":
		e.Status |= 0xB0
	case "progchange":
		e.Status |= 0xC0
	case "chanpressure":
		e.Status |= 0xD0
	case "pitchbend":
		e.Status |= 0xE0
	default:
		log.Printf("SendNoteToSynth: can't handle Note TypeOf=%v\n", note.TypeOf)
		return
	}

	// XXX - Should keep track per-midiOut/per-channel (across all synths) of the current
	// bank/program selected on that midiOut/channel, and if it changes, then (and only then)
	// send the bank/program select

	// if Debug.MIDI {
	// 	log.Printf("Sending portmidi.Event = %s\n", e)
	// }
	if Debug.MIDI {
		log.Printf("SendNoteToSynth: synth=%s status=0x%02x data1=%d data2=%d\n", synth.midiOut.Name(), e.Status, e.Data1, e.Data2)
	}
	synth.midiOut.stream.WriteShort(e.Status, e.Data1, e.Data2)

	// go through Plugins and send the Note
	for _, pluginRef := range oneRouter.plugin {
		if (pluginRef.Events & EventNoteOutput) != 0 {
			log.Printf("Engine is sending note=%s to plugin=%s\n", note, pluginRef.Name)
			pluginRef.forwardToPlugin <- note
		}
	}
}
