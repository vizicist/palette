package kit

import (
	"encoding/json"
	"fmt"
	"sync"

	"gitlab.com/gomidi/midi/v2/drivers"
)

type PortChannel struct {
	Port    string // as used by Portmidi
	Channel int    // 0-15 for MIDI channels 1-16
}

// MIDIChannelOutput is used to remember the last
// bank and program values for a particular output,
// so they're only sent out when they've changed.
type MIDIPortChannelState struct {
	Channel int // 1-based, 1-16
	Bank    int // 1-based, 1-whatever
	Program int // 1-based, 1-128
	PortName string // MIDI output port
	Output  drivers.Out
	Isopen  bool
	Mutex   sync.Mutex
}

var PortChannels map[PortChannel]*MIDIPortChannelState

type Synth struct {
	name          string
	portchannel   PortChannel
	bank          int // 0 if not set
	program       int // 0 if note set
	noteMutex     sync.RWMutex
	noteDown      []bool
	noteDownCount []int
	state         *MIDIPortChannelState
}

var Synths map[string]*Synth

func InitSynths() {

	Synths = make(map[string]*Synth)

	bytes, err := TheHost.GetConfigFileData("synths.json")
	if err != nil {
		LogIfError(err)
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
	LogIfError(json.Unmarshal(bytes, &jsynths))

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
		LogIfError(fmt.Errorf("InitSynths: no default synth in Synths.json, using fake default"))
		Synths["default"] = NewSynth("default", "default", 1, 0, 0)
	}

	LogInfo("Synths loaded", "len", len(Synths))
}

func (synth *Synth) Channel() uint8 {
	return uint8(synth.portchannel.Channel)
}

func (synth *Synth) ClearNoteDowns() {

	synth.noteMutex.Lock()
	defer synth.noteMutex.Unlock()

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

	portchannel := PortChannel{Port: port, Channel: channel}

	var state *MIDIPortChannelState
	if name == "fake" {
		state = TheHost.OpenFakeChannelOutput(port, channel)
	} else {
		state = TheHost.OpenChannelOutput(portchannel)
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

func (state *MIDIPortChannelState) UpdateBankProgram(synth *Synth) {

	state.Mutex.Lock()
	defer state.Mutex.Unlock()

	// LogInfo("Checking Bank Program", "bank", bank, "program", program, "mc.bank", state.bank, "mc.program", state.program, "mc", fmt.Sprintf("%p", state))
	if state.Bank != synth.bank {
		LogWarn("SendBankProgram: XXX - SHOULD be sending", "bank", synth.bank)
		state.Bank = synth.bank
	}
	// if the requested program doesn't match the current one, send it
	if state.Program != synth.program {
		// LogInfo("PROGRAM CHANGED", "program", program, "mc.program", state.program)
		state.Program = synth.program
		status := byte(int64(ProgramStatus) | int64(state.Channel-1))
		data1 := byte(synth.program - 1)
		LogInfo("SendBankProgram: MIDI", "status", hexString(status), "program", hexString(data1))
		LogOfType("midi", "Raw MIDI Output, BankProgram",
			"synth", synth.name,
			"status", "0x"+hexString(status),
			"data1", "0x"+hexString(data1))

		// TheHost.SendMIDI([]byte{status, data1})
		err := state.Output.Send([]byte{status, data1})
		LogIfError(err)
	}
}



func (synth *Synth) updatePortChannelState() (*MIDIPortChannelState, error) {
	state, err := TheHost.GetPortChannelState(synth.portchannel)
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
		LogIfError(err)
		return
	}
	synth.ClearNoteDowns()

	// send All Notes Off
	status := 0xb0 | byte(synth.portchannel.Channel-1)
	data1 := byte(0x7b)
	data2 := byte(0x00)
	LogOfType("midi", "Raw MIDI Output, ANO",
		"synth", synth.name,
		"status", "0x"+hexString(status),
		"data1", "0x"+hexString(data1),
		"data2", "0x"+hexString(data2))

	err = state.Output.Send([]byte{status, data1, data2})
	LogIfError(err)
}

func (synth *Synth) SendController(cnum uint8, cval uint8) {

	state, err := synth.updatePortChannelState()
	if err != nil {
		LogIfError(err)
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
	status := 0xb0 | byte(synth.portchannel.Channel-1)
	data1 := byte(cnum)
	data2 := byte(cval)

	LogOfType("midicontroller", "Raw MIDI Output, controller",
		"synth", synth.name,
		"status", "0x"+hexString(status),
		"data1", "0x"+hexString(data1),
		"data2", "0x"+hexString(data2))

	LogIfError(state.Output.Send([]byte{status, data1, data2}))
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
		velocity = v.Velocity // could be 0, to be interpreted as a NoteOff by receivers

	case *NoteOff:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity

	default:
		LogWarn("SendNoteToMidiOutput: doesn't handle", "type", fmt.Sprintf("%T", v))
		return
	}

	_, err := synth.updatePortChannelState()
	if err != nil {
		LogIfError(err)
		return
	}

	status := byte(synth.portchannel.Channel - 1)
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

		synth.noteMutex.Lock()

		synth.noteDown[pitch] = true
		synth.noteDownCount[pitch]++
		downCount := synth.noteDownCount[pitch]

		synth.noteMutex.Unlock()

		LogOfType("note", "SendNoteOnToSynth",
			"synth", synth,
			"channel", synth.portchannel.Channel,
			"pitch", pitch,
			"notedowncount", downCount)

	case *NoteOff:
		status |= NoteOffStatus
		data2 = 0

		synth.noteMutex.Lock()

		synth.noteDown[pitch] = false
		synth.noteDownCount[pitch]--
		downCount := synth.noteDownCount[pitch]

		synth.noteMutex.Unlock()

		LogOfType("note", "SendNoteOffToSynth",
			"synth", synth,
			"channel", synth.portchannel.Channel,
			"pitch", pitch,
			"notedowncount", downCount)

	default:
		LogWarn("SendNoteToMidiOutput: can't handle", "type", fmt.Sprintf("%T", value))
		return
	}

	synth.SendBytesToMidiOutput([]byte{status, data1, data2})
}

// SendBytesToMidiOutput
func (synth *Synth) SendBytesToMidiOutput(bytes []byte) {

	if len(bytes) == 0 {
		LogWarn("SendBytesToMidiOutput: 0-length bytes?")
		return
	}

	state, err := synth.updatePortChannelState()
	if err != nil {
		LogIfError(err)
		return
	}

	// Use status value from bytes, but channel gets taken from Synth
	status := (bytes[0] & 0xf0) | byte(synth.portchannel.Channel-1)
	bytes[0] = status

	switch len(bytes) {
	case 1:
		LogOfType("midi", "Raw MIDI Output",
			"synth", synth.name,
			"bytes[0]", "0x"+hexString(bytes[0]))
	case 2:
		LogOfType("midi", "Raw MIDI Output",
			"synth", synth.name,
			"bytes[0]", "0x"+hexString(bytes[0]),
			"bytes[1]", "0x"+hexString(bytes[1]))
	case 3:
		LogOfType("midi", "Raw MIDI Output",
			"synth", synth.name,
			"bytes[0]", "0x"+hexString(bytes[0]),
			"bytes[1]", "0x"+hexString(bytes[1]),
			"bytes[2]", "0x"+hexString(bytes[2]))
	default:
		LogOfType("midi", "Raw MIDI Output",
			"synth", synth.name,
			"length", len(bytes),
			"bytes", bytes)
	}

	err = state.Output.Send(bytes)
	if err != nil {
		LogWarn("synth.SendBytesToMidiOutputSend", "err", err)
	}
}

func (synth *Synth) SendBytes(bytes []byte) error {
	return synth.state.Output.Send(bytes)
}

func (synth *Synth) UpdateBankProgram() {

	state := synth.state
	state.Mutex.Lock()
	defer state.Mutex.Unlock()

	// LogInfo("Checking Bank Program", "bank", bank, "program", program, "mc.bank", state.bank, "mc.program", state.program, "mc", fmt.Sprintf("%p", state))
	if state.Bank != synth.bank {
		LogWarn("SendBankProgram: XXX - SHOULD be sending", "bank", synth.bank)
		state.Bank = synth.bank
	}
	// if the requested program doesn't match the current one, send it
	if state.Program != synth.program {
		// LogInfo("PROGRAM CHANGED", "program", program, "mc.program", state.program)
		state.Program = synth.program
		status := byte(int64(ProgramStatus) | int64(state.Channel-1))
		data1 := byte(synth.program - 1)
		LogInfo("SendBankProgram: MIDI", "status", hexString(status), "program", hexString(data1))
		LogOfType("midi", "Raw MIDI Output, BankProgram",
			"synth", synth.name,
			"status", "0x"+hexString(status),
			"data1", "0x"+hexString(data1))

		err := state.Output.Send([]byte{status, data1})
		LogIfError(err)
	}
}
