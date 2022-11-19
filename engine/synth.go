package engine

import (
	"encoding/json"
	"fmt"
	"os"
	// "github.com/vizicist/portmidi"
)

type PortChannel struct {
	port    string // as used by Portmidi
	channel int    // 0-15 for MIDI channels 1-16
}

var PortChannels map[PortChannel]*MIDIChannelOutput

type Synth struct {
	portchannel PortChannel
	bank        int // 0 if not set
	program     int // 0 if note set
	// midiChannelOut *MidiChannelOutput
	noteDown      []bool
	noteDownCount []int
}

var Synths map[string]*Synth

func InitSynths() {

	Synths = make(map[string]*Synth)

	filename := ConfigFilePath("synths.json")
	bytes, err := os.ReadFile(filename)
	if err != nil {
		LogError(err)
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
	Info("Synths loaded", "len", len(Synths))
}

func (synth *Synth) ClearNoteDowns() {
	for i := range synth.noteDown {
		synth.noteDown[i] = false
		synth.noteDownCount[i] = 0
	}
}

func NewSynth(port string, channel int, bank int, program int) *Synth {

	synthoutput := ConfigBoolWithDefault("generatesound", true)

	// If there's already a Synth for this PortChannel, error

	portchannel := PortChannel{port: port, channel: channel}

	var midiChannelOut *MIDIChannelOutput
	if synthoutput {
		midiChannelOut = MIDI.openChannelOutput(portchannel)
	} else {
		midiChannelOut = MIDI.openFakeChannelOutput(port, channel)
	}

	if midiChannelOut == nil {
		Warn("InitSynths: Unable to open", "port", port)
		return nil
	}
	sp := &Synth{
		portchannel: portchannel,
		bank:        bank,
		program:     program,
		// midiChannelOut: midiChannelOut,
		noteDown:      make([]bool, 128),
		noteDownCount: make([]int, 128),
	}
	sp.ClearNoteDowns() // debugging, shouldn't be needed
	return sp
}

// SendANO sends all-notes-off
func SendANOToSynth(synthName string) {
	synth, ok := Synths[synthName]
	if !ok {
		Warn("SendANOToSynth: no such", "synth", synthName)
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

	synth.ClearNoteDowns()
	// for i := range synth.noteDown {
	// 	synth.noteDown[i] = false
	// }
	Warn("SendANOToSynth needs work")
	/*
		status := 0xb0 | (synth.portchannel.channel - 1)
		mc.midiDeviceOutput.stream.WriteShort(int64(status), int64(0x7b), int64(0x00))
	*/
}

func SendControllerToSynth(synthName string, cnum int, cval int) {
	synth, ok := Synths[synthName]
	if !ok {
		Warn("SendControllerToSynth: no such", "synth", synthName)
		return
	}
	mc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	Warn("SendControlToSynth needs work")
	/*
		e := portmidi.Event{
			Timestamp: time.Now(),
			Status:    int64(synth.portchannel.channel - 1),
			Data1:     int64(cnum),
			Data2:     int64(cval),
		}
		e.Status |= 0xb0
		if e.Data2 > 0x7f {
			Warn("SendControllerToSynth: Hey! Data2 shouldn't be > 0x7f")
		} else {
			mc.midiDeviceOutput.stream.WriteShort(e.Status, e.Data1, e.Data2)
		}
	*/
}

// SendNote sends MIDI output for a Note
func SendNoteToSynth(note *Note) {
	synthName := note.Synth
	if synthName == "" {
		synthName = "default"
	}
	synth, ok := Synths[synthName]
	if !ok {
		Warn("SendNoteToSynth: no such", "synth", synthName)
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

	status := byte(synth.portchannel.channel - 1)
	data1 := byte(note.Pitch)
	data2 := byte(note.Velocity)
	if note.TypeOf == "noteon" && note.Velocity == 0 {
		Info("MIDIIO.SendNote: noteon with velocity==0 is changed to a noteoff")
		note.TypeOf = "noteoff"
	}
	switch note.TypeOf {
	case "noteon":
		status |= 0x90

		// We now allow multiple notes with the same pitch,
		// which assumes the synth handles it okay.
		// There might need to be an option to
		// automatically send a noteOff before sending the noteOn.
		// if synth.noteDown[note.Pitch] {
		//     Warn("SendNoteToSynth: Ignoring second noteon")
		// }

		synth.noteDown[note.Pitch] = true
		synth.noteDownCount[note.Pitch]++

		DebugLogOfType("midi", "SendNoteOnToSynth",
			"synth", synth,
			"channel", synth.portchannel.channel,
			"pitch", note.Pitch,
			"notedowncount", synth.noteDownCount[note.Pitch])

	case "noteoff":
		status |= 0x80
		data2 = 0
		synth.noteDown[note.Pitch] = false
		synth.noteDownCount[note.Pitch]--

		DebugLogOfType("midi", "SendNoteOffToSynth",
			"synth", synth,
			"channel", synth.portchannel.channel,
			"pitch", note.Pitch,
			"notedowncount", synth.noteDownCount[note.Pitch])

	case "controller":
		status |= 0xB0
	case "progchange":
		status |= 0xC0
	case "chanpressure":
		status |= 0xD0
	case "pitchbend":
		status |= 0xE0
	default:
		Warn("SendNoteToSynth: can't handle Note TypeOf=%v\n", note.TypeOf)
		return
	}

	DebugLogOfType("midi", "Raw MIDI Output",
		"synth", synth,
		"status", hexString(status),
		"data1", hexString(data1),
		"data2", hexString(data2))

	Info("SendNote:", "status", hexString(status), "data1", hexString(data1), "data2", hexString(data2))
	err := mc.output.Send([]byte{status, data1, data2})
	if err != nil {
		Warn("output.Send", "err", err)
	}
	// mc.midiDeviceOutput.stream.WriteShort(e.Status, e.Data1, e.Data2)
}

func hexString(b byte) string {
	return fmt.Sprintf("%02x", b)
}
