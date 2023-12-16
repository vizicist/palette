package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type PortChannel struct {
	port    string // as used by Portmidi
	channel int    // 0-15 for MIDI channels 1-16
}

var PortChannels map[PortChannel]*MidiPortChannelState

type Synth struct {
	name          string
	portchannel   PortChannel
	bank          int // 0 if not set
	program       int // 0 if note set
	noteMutex     sync.RWMutex
	noteDown      []bool
	noteDownCount []int
	state         *MidiPortChannelState
}

var Synths map[string]*Synth

func InitSynths() {

	LogInfo("InitSynths called")
	Synths = make(map[string]*Synth)

	Synths[""] = OpenNewSynth("", "dummyport", 1, 0, 0)

	filename := ConfigFilePath("synths.json")
	bytes, err := os.ReadFile(filename)
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

		Synths[nm] = OpenNewSynth(nm, port, channel, bank, program)
	}

	LogInfo("Synths loaded", "len", len(Synths))
}

func (synth *Synth) midiOutputEnabled() bool {
	if synth.state == nil {
		LogWarn("Synth.MidiOutputEnabled: state is nil")
		return false
	}
	if synth.state.output == nil {
		LogWarn("Synth.MidiOutputEnabled: output is nil")
		return false
	}
	return true
}

// SendANO sends all-notes-off
func (synth *Synth) SendANO() {

	if !synth.midiOutputEnabled() {
		return
	}

	state, err := synth.updatePortChannelState()
	if err != nil {
		LogIfError(err)
		return
	}
	synth.clearNoteDowns()

	// send All Notes Off
	status := 0xb0 | byte(synth.portchannel.channel-1)
	data1 := byte(0x7b)
	data2 := byte(0x00)
	LogOfType("midi", "Raw MIDI Output, ANO",
		"synth", synth.name,
		"status", "0x"+hexString(status),
		"data1", "0x"+hexString(data1),
		"data2", "0x"+hexString(data2))

	LogIfError(state.output.Send([]byte{status, data1, data2}))
}

func (synth *Synth) SendController(cnum uint8, cval uint8) {

	if !synth.midiOutputEnabled() {
		return
	}

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
	status := 0xb0 | byte(synth.portchannel.channel-1)
	data1 := byte(cnum)
	data2 := byte(cval)

	LogOfType("midicontroller", "Raw MIDI Output, controller",
		"synth", synth.name,
		"status", "0x"+hexString(status),
		"data1", "0x"+hexString(data1),
		"data2", "0x"+hexString(data2))

	LogIfError(state.output.Send([]byte{status, data1, data2}))
}

func (synth *Synth) SendNoteToMidiOutput(value any) {

	if !synth.midiOutputEnabled() {
		return
	}

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

		synth.noteMutex.Lock()

		synth.noteDown[pitch] = true
		synth.noteDownCount[pitch]++
		downCount := synth.noteDownCount[pitch]

		synth.noteMutex.Unlock()

		LogOfType("note", "SendNoteOnToSynth",
			"synth", synth,
			"channel", synth.portchannel.channel,
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
			"channel", synth.portchannel.channel,
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

	if !synth.midiOutputEnabled() {
		return
	}

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
	status := (bytes[0] & 0xf0) | byte(synth.portchannel.channel-1)
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

	err = state.output.Send(bytes)
	if err != nil {
		LogWarn("synth.SendBytesToMidiOutputSend", "err", err)
	}
}

func (synth *Synth) Channel() uint8 {
	return uint8(synth.portchannel.channel)
}

func (synth *Synth) clearNoteDowns() {

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
		LogWarn("GetSynth: no Synth, using empty Synth", "synthName", synthName)
		return Synths[""]
	}
	return synth
}

var FakeIndex int

func OpenNewSynth(name string, port string, channel int, bank int, program int) *Synth {

	// XXX - If there's already a Synth for this PortChannel, should error
	portchannel := PortChannel{port: port, channel: channel}

	state := TheMidiIO.NewPortChannelState(portchannel)
	return NewSynth(name, portchannel, bank, program, state)
}

/*
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
	sp.clearNoteDowns() // debugging, shouldn't be needed
	return sp
}
*/

func NewSynth(name string, portchannel PortChannel, bank int, program int, state *MidiPortChannelState) *Synth {
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
	sp.clearNoteDowns() // debugging, shouldn't be needed
	return sp
}

func (synth *Synth) updatePortChannelState() (*MidiPortChannelState, error) {
	if TheMidiIO == nil {
		return nil, fmt.Errorf("updatePortChannelState: TheMidiIO is nil?")
	}
	state, err := TheMidiIO.GetPortChannelState(synth.portchannel)
	if err != nil {
		return nil, err
	}
	// This only sends the bank and/or program if they change
	synth.updateBankProgram()
	return state, nil
}

func (synth *Synth) updateBankProgram() {

	state := synth.state
	state.mutex.Lock()
	defer state.mutex.Unlock()

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

		LogIfError(state.output.Send([]byte{status, data1}))
	}
}

func hexString(b byte) string {
	return fmt.Sprintf("%02x", b)
}
