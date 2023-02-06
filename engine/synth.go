package engine

import (
	"encoding/json"
	"fmt"
	"os"
)

type PortChannel struct {
	port    string // as used by Portmidi
	channel int    // 0-15 for MIDI channels 1-16
}

var PortChannels map[PortChannel]*MIDIPortChannelState

type Synth struct {
	name        string
	portchannel PortChannel
	bank        int // 0 if not set
	program     int // 0 if note set
	// midiChannelOut *MidiChannelOutput
	noteDown      []bool
	noteDownCount []int
	state         *MIDIPortChannelState
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
	LogError(json.Unmarshal(bytes, &jsynths))

	for i := range jsynths.Synths {
		nm := jsynths.Synths[i].Name
		port := jsynths.Synths[i].Port
		channel := jsynths.Synths[i].Channel
		bank := jsynths.Synths[i].Bank
		program := jsynths.Synths[i].Program

		Synths[nm] = NewSynth(nm, port, channel, bank, program)
	}

	_, ok := Synths["default"]
	if !ok {
		LogError(fmt.Errorf("InitSynths: no default synth in Synths.json, using fake default"))
		Synths["default"] = NewSynth("default", "default", 1, 0, 0)
	}

	LogInfo("Synths loaded", "len", len(Synths))
}

func (synth *Synth) Channel() uint8 {
	return uint8(synth.portchannel.channel)
}

func (synth *Synth) ClearNoteDowns() {
	for i := range synth.noteDown {
		synth.noteDown[i] = false
		synth.noteDownCount[i] = 0
	}
}

func GetSynth(synthName string) *Synth {
	synth, ok := Synths[synthName]
	if !ok {
		return nil
	}
	return synth
}

func NewSynth(name string, port string, channel int, bank int, program int) *Synth {

	// If there's already a Synth for this PortChannel, should error

	portchannel := PortChannel{port: port, channel: channel}

	var state *MIDIPortChannelState
	if name == "fake" {
		state = MIDI.openFakeChannelOutput(port, channel)
	} else {
		state = MIDI.openChannelOutput(portchannel)
	}

	if state == nil {
		LogWarn("InitSynths: Unable to open", "port", port)
		return nil
	}
	sp := &Synth{
		name:        name,
		portchannel: portchannel,
		bank:        bank,
		program:     program,
		// midiChannelOut: midiChannelOut,
		noteDown:      make([]bool, 128),
		noteDownCount: make([]int, 128),
		state:         state,
	}
	sp.ClearNoteDowns() // debugging, shouldn't be needed
	return sp
}

func (synth *Synth) updatePortChannelState() (*MIDIPortChannelState, error) {
	state, err := MIDI.GetPortChannelState(synth.portchannel)
	if err != nil {
		return nil, err
	}
	// This only sends the bank and/or program if they change
	synth.UpdateBankProgram()
	return state, nil
}

// SendANO sends all-notes-off
func (synth *Synth) SendANO() {

	state, err := synth.updatePortChannelState()
	if err != nil {
		LogError(err)
		return
	}
	synth.ClearNoteDowns()

	// send All Notes Off
	status := 0xb0 | byte(synth.portchannel.channel-1)
	data1 := byte(0x7b)
	data2 := byte(0x00)
	LogOfType("midi", "Raw MIDI Output, ANO",
		"synth", synth.name,
		"status", "0x"+hexString(status),
		"data1", "0x"+hexString(data1),
		"data2", "0x"+hexString(data2))

	LogError(state.output.Send([]byte{status, data1, data2}))
}

func (synth *Synth) SendController(cnum int, cval int) {

	state, err := synth.updatePortChannelState()
	if err != nil {
		LogError(err)
		return
	}

	if cnum > 0x7f {
		LogWarn("SendControllerToSynth: invalid value", "cnum", cnum)
		return
	}
	if cval > 0x7f {
		LogWarn("SendControllerToSynth: invalid value", "cval", cval)
		return
	}
	status := 0xb0 | byte(synth.portchannel.channel-1)
	data1 := byte(cnum)
	data2 := byte(cval)

	LogOfType("midi", "Raw MIDI Output, controller",
		"synth", synth.name,
		"status", "0x"+hexString(status),
		"data1", "0x"+hexString(data1),
		"data2", "0x"+hexString(data2))

	LogError(state.output.Send([]byte{status, data1, data2}))
}

/*
func SendToSynth(value any) {

	var channel uint8
	var pitch uint8
	var velocity uint8

	switch v := value.(type) {
	case *NoteOn:
		channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
		if velocity == 0 {
			LogInfo("MIDIIO.SendNote: noteon with velocity==0 NOT changed to a noteoff")
		}
	case *NoteOff:
		channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteFull:
		channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	default:
		LogWarn("SendToSynth: doesn't handle", "type", fmt.Sprintf("%T", v))
		return
	}

	synthName := fmt.Sprintf("P_01_C_%02d", channel+1)
	synth, ok := Synths[synthName]
	if !ok {
		LogWarn("SendPhraseElementToSynth: no such", "synth", synthName)
		return
	}

	if synth == nil {
		// We don't complain, we assume the inability to open the
		// synth named synthName has already been logged.
		return
	wsc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	status := byte(synth.portchannel.channel - 1)
	data1 := pitch
	data2 := velocity
	switch value.(type) {
	case *NoteOn:
		status |= NoteOnStatus

		// We now allow multiple notes with the same pitch,
		// which assumes the synth handles it okay.
		// There might need to be an option to
		// automatically send a noteOff before sending the noteOn.
		// if synth.noteDown[note.Pitch] {
		//     Warn("SendPhraseElementToSynth: Ignoring second noteon")
		// }

		synth.noteDown[pitch] = true
		synth.noteDownCount[pitch]++

		LogOfType("midi", "SendNoteOnToSynth",
			"synth", synth,
			"channel", synth.portchannel.channel,
			"pitch", pitch,
			"notedowncount", synth.noteDownCount[pitch])

	case *NoteOff:
		status |= NoteOffStatus
		data2 = 0
		synth.noteDown[pitch] = false
		synth.noteDownCount[pitch]--

		LogOfType("midi", "SendNoteOffToSynth",
			"synth", synth,
			"channel", synth.portchannel.channel,
			"pitch", pitch,
			"notedowncount", synth.noteDownCount[pitch])

	// case "controller":
	// 	status |= 0xB0
	// case "progchange":
	// 	status |= 0xC0
	// case "chanpressure":
	// 	status |= 0xD0
	// case "pitchbend":
	// 	status |= 0xE0

	default:
		LogWarn("SendToSynth: can't handle", "type", fmt.Sprintf("%T", value))
		return
	}

	LogOfType("midi", "Raw MIDI Output",
		"synth", synth,
		"status", hexString(status),
		"data1", hexString(data1),
		"data2", hexString(data2))

	err := mc.output.Send([]byte{status, data1, data2})
	if err != nil {
		LogWarn("output.Send", "err", err)
	}
}
*/

func hexString(b byte) string {
	return fmt.Sprintf("%02x", b)
}

func (synth *Synth) SendNoteToMidiOutput(value any) {

	// var channel uint8
	var pitch uint8
	var velocity uint8

	switch v := value.(type) {

	case *NoteOn:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
		if velocity == 0 {
			LogInfo("MIDIIO.SendNote: noteon with velocity==0 NOT changed to a noteoff")
		}

	case *NoteOff:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity

	// case *NoteFull:
	// 	// channel = v.Channel
	// 	pitch = v.Pitch
	// 	velocity = v.Velocity

	default:
		LogWarn("SendToSynth: doesn't handle", "type", fmt.Sprintf("%T", v))
		return
	}

	state, err := synth.updatePortChannelState()
	if err != nil {
		LogError(err)
		return
	}

	status := byte(synth.portchannel.channel - 1)
	data1 := pitch
	data2 := velocity
	switch value.(type) {
	case *NoteOn:
		status |= NoteOnStatus

		// We now allow multiple notes with the same pitch,
		// which assumes the synth handles it okay.
		// There might need to be an option to
		// automatically send a noteOff before sending the noteOn.
		// if synth.noteDown[note.Pitch] {
		//     Warn("SendPhraseElementToSynth: Ignoring second noteon")
		// }

		synth.noteDown[pitch] = true
		synth.noteDownCount[pitch]++

		LogOfType("note", "SendNoteOnToSynth",
			"synth", synth,
			"channel", synth.portchannel.channel,
			"pitch", pitch,
			"notedowncount", synth.noteDownCount[pitch])

	case *NoteOff:
		status |= NoteOffStatus
		data2 = 0
		synth.noteDown[pitch] = false
		synth.noteDownCount[pitch]--

		LogOfType("note", "SendNoteOffToSynth",
			"synth", synth,
			"channel", synth.portchannel.channel,
			"pitch", pitch,
			"notedowncount", synth.noteDownCount[pitch])

	// case "controller":
	// 	status |= 0xB0
	// case "progchange":
	// 	status |= 0xC0
	// case "chanpressure":
	// 	status |= 0xD0
	// case "pitchbend":
	// 	status |= 0xE0

	default:
		LogWarn("SendToSynth: can't handle", "type", fmt.Sprintf("%T", value))
		return
	}

	LogOfType("midi", "Raw MIDI Output",
		"synth", synth.name,
		"status", "0x"+hexString(status),
		"data1", "0x"+hexString(data1),
		"data2", "0x"+hexString(data2))

	err = state.output.Send([]byte{status, data1, data2})
	if err != nil {
		LogWarn("output.Send", "err", err)
	}
}

func (synth *Synth) SendBytes(bytes []byte) error {
	return synth.state.output.Send(bytes)
}

func (synth *Synth) UpdateBankProgram() {

	state := synth.state
	synth.state.mutex.Lock()
	defer synth.state.mutex.Unlock()

	// LogInfo("Checking Bank Program", "bank", bank, "program", program, "mc.bank", state.bank, "mc.program", state.program, "mc", fmt.Sprintf("%p", state))
	if state.bank != synth.bank {
		LogWarn("SendBankProgram: XXX - SHOULD be sending", "bank", synth.bank)
		state.bank = synth.bank
	}
	// if the requested program doesn't match the current one, send it
	if state.program != synth.program {
		// LogInfo("PROGRAM CHANGED", "program", program, "mc.program", state.program)
		state.program = synth.program
		status := byte(int64(ProgramStatus) | int64(state.channel-1))
		data1 := byte(synth.program - 1)
		LogInfo("SendBankProgram: MIDI", "status", hexString(status), "program", hexString(data1))
		LogOfType("midi", "Raw MIDI Output, BankProgram",
			"synth", synth.name,
			"status", "0x"+hexString(status),
			"data1", "0x"+hexString(data1))

		LogError(state.output.Send([]byte{status, data1}))
	}
}
