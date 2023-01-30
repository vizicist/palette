//go:build windows
// +build windows

package engine

import (
	"fmt"
	"strings"
	"sync"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

// These are the values of MIDI status bytes
const (
	// NoteOn
	NoteOnStatus       byte = 0x90
	NoteOffStatus      byte = 0x80
	PressureStatus     byte = 0xa0
	ControllerStatus   byte = 0xb0
	ProgramStatus      byte = 0xc0
	ChanPressureStatus byte = 0xd0
	PitchbendStatus    byte = 0xe0
)

// MIDIIO encapsulate everything having to do with MIDI I/O
type MIDIIO struct {
	midiInput            drivers.In
	midiOutputs          map[string]drivers.Out // synth name is the key
	midiPortChannelState map[PortChannel]*MIDIPortChannelState
	midiInputChan        chan MidiEvent
	stop                 func()
}

// MIDIChannelOutput is used to remember the last
// bank and program values for a particular output,
// so they're only sent out when they've changed.
type MIDIPortChannelState struct {
	channel int // 1-based, 1-16
	bank    int // 1-based, 1-whatever
	program int // 1-based, 1-128
	output  drivers.Out
	isopen  bool
	mutex   sync.Mutex
}

func DefaultMIDIChannelOutput() *MIDIPortChannelState {
	return &MIDIPortChannelState{}
}

type MidiEvent struct {
	Msg midi.Message
}

func MidiEventFromMap(args map[string]string) (MidiEvent, error) {

	msg, ok := args["msg"]
	if !ok {
		return MidiEvent{}, fmt.Errorf("missing msg argument")
	}
	_ = msg
	return MidiEvent{}, fmt.Errorf("MakeMidiEvent needs work")
	// var me MidiEvent
	// me.Msg = midi.MessageFromString(msg)
}

func (me MidiEvent) ToMap() map[string]string {
	return map[string]string{
		"event": "midi",
		"msg":   me.Msg.String(),
	}
}

// MIDI is a pointer to
var MIDI *MIDIIO

// InitMIDI initializes stuff
func InitMIDI() {

	InitializeClicksPerSecond(defaultClicksPerSecond)

	MIDI = &MIDIIO{
		midiInput:            nil,
		midiOutputs:          make(map[string]drivers.Out),
		midiPortChannelState: make(map[PortChannel]*MIDIPortChannelState),
		stop:                 nil,
	}

	// util.InitScales()

	// erae := false
	inports := midi.GetInPorts()
	outports := midi.GetOutPorts()

	midiInputName := EngineParam("midiinput")

	// device names is done with strings.Contain	// Note: name matching of MIDI s
	// beause gomidi/midi appends an index value to the strings

	for _, inp := range inports {
		name := inp.String()
		LogOfType("midiports", "MIDI input", "port", name)
		// if strings.Contains(name, "Erae Touch") {
		// 	erae = true
		// 	EraeInput = inp
		// }
		if strings.Contains(name, midiInputName) {
			// We only open a single input, though midiInputs is an array
			MIDI.midiInput = inp
		}
	}

	for _, outp := range outports {
		name := outp.String()
		// NOTE: name is the port name followed by an index
		LogOfType("midiports", "MIDI output", "port", outp.String())
		// if strings.Contains(name, "Erae Touch") {
		// 	EraeOutput = outp
		// }
		MIDI.midiOutputs[name] = outp
	}

	LogInfo("Initialized MIDI", "outports", outports, "midiInputName", midiInputName)

	// if erae {
	// 	Info("Erae Touch input is being enabled")
	// 	InitErae()
	// }
}

type MidiHandlerFunc func(midi.Message, int32)

func (m *MIDIIO) Start(inChan chan MidiEvent) {
	m.midiInputChan = inChan
	stop, err := midi.ListenTo(m.midiInput, m.handleMidiInput, midi.UseSysEx())
	if err != nil {
		LogWarn("midi.ListenTo", "err", err)
		return
	}
	m.stop = stop

	select {}
}

func (m *MIDIIO) handleMidiInput(msg midi.Message, timestamp int32) {
	var bt []byte
	var ch, key, vel uint8
	LogOfType("midi", "handleMidiInput", "msg", msg)
	switch {
	case msg.GetSysEx(&bt):
		LogOfType("midi", "midi.ListenTo sysex", "bt", bt)

	case msg.GetNoteOn(&ch, &key, &vel):
		LogOfType("midi", "midi.ListenTo notestart", "bt", bt)
		m.midiInputChan <- MidiEvent{Msg: msg}

	case msg.GetNoteOff(&ch, &key, &vel):
		LogOfType("midi", "midi.ListenTo noteend", "bt", bt)
		m.midiInputChan <- MidiEvent{Msg: msg}
	default:
		LogWarn("Unable to handle MIDI input", "msg", msg)
	}
}

/*
func (m* EraeWriteSysEx(bytes []byte) {
	EraeMutex.Lock()
	defer EraeMutex.Unlock()
	EraeOutput.midiDeviceOutput.WriteSysEx(bytes)
}
*/

func (state *MIDIPortChannelState) UpdateBankProgram(synth *Synth) {

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

		LogError(state.output.Send([]byte{status, data1}))
	} else {
		// LogInfo("PROGRAM DID NOT CHANGE", "program", program, "mc.program", state.program)
	}
}

func (out *MIDIPortChannelState) WriteShort(status, data1, data2 int64) {
	LogWarn("WriteShort needs work")
}

func (m *MIDIIO) GetPortChannelState(portchannel PortChannel) (*MIDIPortChannelState, error) {
	mc, ok := m.midiPortChannelState[portchannel]
	if !ok {
		return nil, fmt.Errorf("GetMidiChannelOutput: no entry, port=%s channel=%d", portchannel.port, portchannel.channel)
	}
	if mc.output == nil {
		return nil, fmt.Errorf("GetMidiChannelOutput: midiDeviceOutput==nil, port=%s channel=%d", portchannel.port, portchannel.channel)
	}
	if !mc.isopen {
		e := mc.output.Open()
		if e != nil {
			return nil, e
		}
	}
	mc.isopen = true
	return mc, nil
}

func (m *MIDIIO) openChannelOutput(portchannel PortChannel) *MIDIPortChannelState {

	portName := portchannel.port
	// The portName in portchannel is from Synths.json, while
	// names in midiOutputs are from the gomidi/midi package.
	// For example, portchannel.port might be "01. Internal MIDI",
	// while the gomidi/midi name might be "01. Internal MIDI 1".
	// We assume that the Synths.json value is contained in the
	// drivers.Out name, and we search for that.
	var output drivers.Out = nil
	for name, outp := range m.midiOutputs {
		if strings.Contains(name, portName) {
			output = outp
			break
		}
	}
	if output == nil {
		LogWarn("No output found matching", "port", portName)
		return nil
	}

	state := &MIDIPortChannelState{
		channel: portchannel.channel,
		bank:    0,
		program: 0,
		output:  output,
		isopen:  false,
	}
	m.midiPortChannelState[portchannel] = state
	return state
}

func (m *MIDIIO) openFakeChannelOutput(port string, channel int) *MIDIPortChannelState {

	co := &MIDIPortChannelState{
		channel: channel,
		bank:    0,
		program: 0,
		output:  nil,
		isopen:  false,
	}
	return co
}

func (me MidiEvent) Status() uint8 {
	bytes := me.Msg.Bytes()
	if len(bytes) == 0 {
		LogWarn("Empty bytes in MidiEvent?")
		return 0
	}
	return bytes[0]
}

func (me MidiEvent) Data1() uint8 {
	bytes := me.Msg.Bytes()
	if len(bytes) < 2 {
		LogWarn("No Data1 byte in MidiEvent?")
		return 0
	}
	return bytes[1]
}
