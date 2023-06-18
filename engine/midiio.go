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

// MidiIO encapsulate everything having to do with MIDI I/O
type MidiIO struct {
	midiInput            drivers.In
	midiOutputs          map[string]drivers.Out // synth name is the key
	midiPortChannelState map[PortChannel]*MIDIPortChannelState
	inports              midi.InPorts
	outports             midi.OutPorts
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

// MIDI is a pointer to
var TheMidiIO *MidiIO

func NewMidiIO() *MidiIO {
	return &MidiIO{
		midiInput:            nil,
		midiOutputs:          make(map[string]drivers.Out),
		midiPortChannelState: make(map[PortChannel]*MIDIPortChannelState),
		stop:                 nil,
		inports:              midi.GetInPorts(),
		outports:             midi.GetOutPorts(),
		// engineTranspose:      0,
		// autoTransposeOn:      false,
		// autoTransposeNext:    0,
		// autoTransposeClicks:  8 * OneBeat,
		// autoTransposeIndex:   0,
		// autoTransposeValues:  []int{0, -2, 3, -5},
	}
}

// InitMIDI initializes stuff
func InitMidiIO() {

	for _, outp := range TheMidiIO.outports {
		name := outp.String()
		// NOTE: name is the port name followed by an index
		LogOfType("midiports", "MIDI output", "port", outp.String())
		if strings.Contains(name, "Erae Touch") {
			TheErae.output = outp
		}
		TheMidiIO.midiOutputs[name] = outp
	}

	LogInfo("Initialized MIDI", "outports", TheMidiIO.outports)

	// if erae {
	// 	Info("Erae Touch input is being enabled")
	// 	InitErae()
	// }
}

func (m *MidiIO) SetMidiInput(midiInputName string) {

	if TheMidiIO.midiInput != nil {
		err := TheMidiIO.midiInput.Close()
		LogIfError(err)
		TheMidiIO.midiInput = nil
	}

	if midiInputName == "" {
		return
	}

	// Note: name matching of MIDI device names is done with strings.Contain
	// beause gomidi/midi appends an index value to the strings

	for _, inp := range m.inports {
		name := inp.String()
		LogOfType("midiports", "MIDI input", "port", name)
		if strings.Contains(name, midiInputName) {
			// We only open a single input, though midiInputs is an array
			LogInfo("Opening MIDI input", "name", name)
			TheMidiIO.midiInput = inp
			break // only pick the first one that matches
		}
	}
}

type MidiHandlerFunc func(midi.Message, int32)

func (m *MidiIO) handleMidiError(err error) {
	LogIfError(err)
}

func (m *MidiIO) Start() {

	if m.midiInput == nil {
		LogWarn("No MIDI input port has been selected")
		return
	}
	stop, err := midi.ListenTo(m.midiInput, m.handleMidiInput, midi.UseSysEx(), midi.HandleError(m.handleMidiError))
	if err != nil {
		LogWarn("midi.ListenTo", "err", err)
		return
	}
	m.stop = stop
	select {} // is this needed?
}

func (m *MidiIO) handleMidiInput(msg midi.Message, timestamp int32) {
	LogOfType("midi", "handleMidiInput", "msg", msg)
	TheRouter.midiInputChan <- NewMidiEvent(CurrentClick(),"handleMidiInput",msg)

	/*
		var bt []byte
		var ch, key, vel uint8
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
	*/
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

		LogIfError(state.output.Send([]byte{status, data1}))
	}
}

func (out *MIDIPortChannelState) WriteShort(status, data1, data2 int64) {
	LogWarn("WriteShort needs work")
}

func (m *MidiIO) GetPortChannelState(portchannel PortChannel) (*MIDIPortChannelState, error) {

	state, ok := m.midiPortChannelState[portchannel]
	if !ok {
		return nil, fmt.Errorf("GetMidiChannelOutput: no entry, port=%s channel=%d", portchannel.port, portchannel.channel)
	}
	if state.output == nil {
		return nil, fmt.Errorf("GetMidiChannelOutput: midiDeviceOutput==nil, port=%s channel=%d", portchannel.port, portchannel.channel)
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()
	if !state.isopen {
		e := state.output.Open()
		if e != nil {
			return nil, e
		}
	}
	state.isopen = true
	return state, nil
}

func (m *MidiIO) openChannelOutput(portchannel PortChannel) *MIDIPortChannelState {

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

func (m *MidiIO) openFakeChannelOutput(port string, channel int) *MIDIPortChannelState {

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

