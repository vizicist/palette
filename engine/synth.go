package engine

import (
	"encoding/json"
	"log"
	"os"

	"github.com/vizicist/portmidi"
)

type PortChannel struct {
	port    string // as used by Portmidi
	channel int    // 0-15 for MIDI channels 1-16
}

var PortChannels map[PortChannel]*MidiChannelOutput

type Synth struct {
	portchannel PortChannel
	bank        int // 0 if not set
	program     int // 0 if note set
	// midiChannelOut *MidiChannelOutput
	noteDown []bool
}

var Synths map[string]*Synth

func InitSynths() {

	Synths = make(map[string]*Synth)

	filename := ConfigFilePath("synths.json")
	bytes, err := os.ReadFile(filename)
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

	for i := range jsynths.Synths {
		nm := jsynths.Synths[i].Name
		port := jsynths.Synths[i].Port
		channel := jsynths.Synths[i].Channel
		bank := jsynths.Synths[i].Bank
		program := jsynths.Synths[i].Program

		Synths[nm] = NewSynth(port, channel, bank, program)
	}
	log.Printf("Synths loaded, len=%d\n", len(Synths))
}

func NewSynth(port string, channel int, bank int, program int) *Synth {

	synthoutput := ConfigBoolWithDefault("generatesound", true)

	// If there's already a Synth for this PortChannel, error

	portchannel := PortChannel{port: port, channel: channel}

	var midiChannelOut *MidiChannelOutput
	if synthoutput {
		midiChannelOut = MIDI.openChannelOutput(portchannel)
	} else {
		midiChannelOut = MIDI.openFakeChannelOutput(port, channel)
	}

	if midiChannelOut == nil {
		log.Printf("InitSynths: Unable to open midi port=%s\n", port)
		return nil
	} else {
		return &Synth{
			portchannel: portchannel,
			bank:        bank,
			program:     program,
			// midiChannelOut: midiChannelOut,
			noteDown: make([]bool, 128),
		}
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
	mc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	status := 0xb0 | (synth.portchannel.channel - 1)
	for i := range synth.noteDown {
		synth.noteDown[i] = false
	}
	// log.Printf("SendANOToSynth: synth=%s\n", synthName)
	mc.midiDeviceOutput.stream.WriteShort(int64(status), int64(0x7b), int64(0x00))
}

func SendControllerToSynth(synthName string, cnum int, cval int) {
	synth, ok := Synths[synthName]
	if !ok {
		log.Printf("SendControllerToSynth: no such synth - %s\n", synthName)
		return
	}
	mc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(synth.portchannel.channel - 1),
		Data1:     int64(cnum),
		Data2:     int64(cval),
	}
	e.Status |= 0xb0
	if Debug.MIDI {
		log.Printf("SendControllerToSynth: synth=%s status=0x%02x data1=%d data2=%d\n", mc.midiDeviceOutput.Name(), e.Status, e.Data1, e.Data2)
	}
	if e.Data2 > 0x7f {
		log.Printf("SendControllerToSynth: Hey! Data2 shouldn't be > 0x7f\n")
	} else {
		mc.midiDeviceOutput.stream.WriteShort(e.Status, e.Data1, e.Data2)
	}
}

// SendNote sends MIDI output for a Note
func SendNoteToSynth(note *Note) {
	synthName := note.Sound
	if synthName == "" {
		synthName = "default"
	}
	synth, ok := Synths[synthName]
	if !ok {
		log.Printf("SendNoteToSynth: no such synth - %s\n", synthName)
		return
	}
	if synth == nil {
		// We don't complain, we assume the inability to open the
		// synth named synthName has already been logged.
		return
	}
	mc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(synth.portchannel.channel - 1),
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
			log.Printf("SendNoteToSynth: Ignoring second noteon for chan=%d pitch=%d\n", synth.portchannel.channel, note.Pitch)
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

	if Debug.MIDI {
		log.Printf("SendNoteToSynth: synth=%s status=0x%02x data1=%d data2=%d\n", mc.midiDeviceOutput.Name(), e.Status, e.Data1, e.Data2)
	}

	mc.midiDeviceOutput.stream.WriteShort(e.Status, e.Data1, e.Data2)

	r := TheRouter()
	if r.publishNote {
		PublishNoteEvent(PaletteOutputEventSubject, note, "engine")
	}
}
