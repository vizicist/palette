//go:build windows
// +build windows

package engine

import (
	"fmt"
	"strings"

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
	midiInput          drivers.In
	midiOutputs        map[string]drivers.Out // synth name is the key
	midiChannelOutputs map[PortChannel]*MIDIChannelOutput
	midiInputChan      chan MidiEvent
	stop               func()
}

// MIDIChannelOutput is used to remember the last
// bank and program values for a particular output,
// so they're only sent out when they've changed.
type MIDIChannelOutput struct {
	channel int // 1-based, 1-16
	bank    int // 1-based, 1-whatever
	program int // 1-based, 1-128
	output  drivers.Out
	isopen  bool
}

func DefaultMIDIChannelOutput() *MIDIChannelOutput {
	return &MIDIChannelOutput{}
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
		midiInput:          nil,
		midiOutputs:        make(map[string]drivers.Out),
		midiChannelOutputs: make(map[PortChannel]*MIDIChannelOutput),
		stop:               nil,
	}

	// util.InitScales()

	// erae := false
	inports := midi.GetInPorts()
	outports := midi.GetOutPorts()

	midiInputName := ConfigValue("midiinput")

	// device names is done with strings.Contain	// Note: name matching of MIDI s
	// beause gomidi/midi appends an index value to the strings

	for _, inp := range inports {
		name := inp.String()
		DebugLogOfType("midiports", "MIDI input", "port", name)
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
		DebugLogOfType("midiports", "MIDI output", "port", outp.String())
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
	DebugLogOfType("midi", "handleMidiInput", "msg", msg)
	switch {
	case msg.GetSysEx(&bt):
		DebugLogOfType("midi", "midi.ListenTo sysex", "bt", bt)

	case msg.GetNoteOn(&ch, &key, &vel):
		DebugLogOfType("midi", "midi.ListenTo notestart", "bt", bt)
		m.midiInputChan <- MidiEvent{Msg: msg}

	case msg.GetNoteOff(&ch, &key, &vel):
		DebugLogOfType("midi", "midi.ListenTo noteend", "bt", bt)
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

func (mc *MIDIChannelOutput) SendBankProgram(bank int, program int) {
	if mc.bank != bank {
		LogWarn("SendBankProgram: XXX - SHOULD be sending", "bank", bank)
		mc.bank = bank
	}
	// if the requested program doesn't match the current one, send it
	if mc.program != program {
		mc.program = program
		status := byte(int64(ProgramStatus) | int64(mc.channel-1))
		data1 := byte(program - 1)
		LogInfo("SendBankProgram: MIDI", "status", hexString(status), "program", hexString(data1))
		mc.output.Send([]byte{status, data1})
	}
}

func (out *MIDIChannelOutput) WriteShort(status, data1, data2 int64) {
	LogWarn("WriteShort needs work")
}

func (m *MIDIIO) GetMidiChannelOutput(portchannel PortChannel) *MIDIChannelOutput {
	mc, ok := m.midiChannelOutputs[portchannel]
	if !ok {
		LogWarn("GetMidiChannelOutput: no entry", "port", portchannel.port, "channel", portchannel.channel)
		return nil
	}
	if mc.output == nil {
		LogWarn("GetMidiChannelOutput: midiDeviceOutput==nil", "port", portchannel.port, "channel", portchannel.channel)
		return nil
	}
	if !mc.isopen {
		e := mc.output.Open()
		if e != nil {
			LogError(e, "output", mc.output.String())
			return nil
		}
	}
	mc.isopen = true
	return mc
}

func (m *MIDIIO) openChannelOutput(portchannel PortChannel) *MIDIChannelOutput {

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

	mc := &MIDIChannelOutput{
		channel: portchannel.channel,
		bank:    0,
		program: 0,
		output:  output,
		isopen:  false,
	}
	m.midiChannelOutputs[portchannel] = mc
	return mc
}

func (m *MIDIIO) openFakeChannelOutput(port string, channel int) *MIDIChannelOutput {

	co := &MIDIChannelOutput{
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
