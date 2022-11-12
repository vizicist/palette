//go:build windows
// +build windows

package engine

import (
	"strings"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

// These are the values of MIDI status bytes
const (
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
	// synth name is the key in these maps
	Input              drivers.In
	midiOutputs        map[string]drivers.Out
	midiChannelOutputs map[PortChannel]*MIDIChannelOutput
	stop               func()
	/*
		// MIDI device name is the key in these maps
		outputDeviceID     map[string]portmidi.DeviceID
		outputDeviceInfo   map[string]*portmidi.DeviceInfo
		outputDeviceStream map[string]*portmidi.Stream
		inputDeviceID      map[string]portmidi.DeviceID
		inputDeviceInfo    map[string]*portmidi.DeviceInfo
		inputDeviceStream  map[string]*portmidi.Stream
	*/
}

type MIDIChannelOutput struct {
	channel int // 1-based, 1-16
	bank    int // 1-based, 1-whatever
	program int // 1-based, 1-128
	output  drivers.Out
	isopen  bool
}

type MidiEvent struct {
	Timestamp time.Time
	Status    int64 // if == 0xF0, use SysEx, otherwise Data1 and Data2.
	Data1     int64
	Data2     int64
	SysEx     []byte
	// source    string // {midiinputname} or maybe {IPaddress}.{midiinputname}
}

// MIDI is a pointer to
var MIDI *MIDIIO

// InitMIDI initializes stuff
func InitMIDI() {

	InitializeClicksPerSecond(defaultClicksPerSecond)

	MIDI = &MIDIIO{
		Input:              nil,
		midiOutputs:        make(map[string]drivers.Out),
		midiChannelOutputs: make(map[PortChannel]*MIDIChannelOutput),
		stop:               nil,
	}

	// util.InitScales()

	erae := false
	inports := midi.GetInPorts()
	outports := midi.GetOutPorts()

	midiInputName := ConfigValue("midiinput")

	// We only open a single input, though midiInputs is an array
	for _, inp := range inports {
		name := inp.String()
		Info("MIDI input", "port", name)
		if name == "Erae Touch" {
			erae = true
			EraeInput = inp
		}
		if name == midiInputName {
			MIDI.Input = inp
		}
	}

	for _, outp := range outports {
		name := outp.String()
		// NOTE: name is the port name followed by an index
		Info("MIDI output", "port", outp.String())
		if strings.Contains(name, "Erae Touch") {
			EraeOutput = outp
		}
		MIDI.midiOutputs[name] = outp
	}

	if erae {
		Info("Erae Touch input is being enabled")
		InitErae()
	}
}

func (m *MIDIIO) Start() {
	if MIDI.Input == nil {
		// Assumes higher-level code chooses whether or not to complain visibly
		return
	}
	stop, err := midi.ListenTo(MIDI.Input, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			Info("got sysex", "bt", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			Info("starting note on channel with velocity\n", "note", midi.Note(key), "channel", ch, "velocity", vel)
		case msg.GetNoteEnd(&ch, &key):
			Info("ending note on channel", "note", midi.Note(key), "channel", ch)
		default:
			// ignore
		}
		// Feed the MIDI bytes to r.MIDIInput one byte at a time
		/*
			for _, event := range events {
				e.Router.MIDIInput <- event
				if e.Router.midiEventHandler != nil {
					e.Router.midiEventHandler(event)
				}
			}
		*/
	}, midi.UseSysEx())

	if err != nil {
		LogError(err)
		return
	}
	MIDI.stop = stop

	select {}
}

// func (mco *MIDIChannelOutput) xWriteSysEx(bb []byte) {
// 	mco.WriteSysEx(bb)
// }

/*
func (m* EraeWriteSysEx(bytes []byte) {
	EraeMutex.Lock()
	defer EraeMutex.Unlock()
	EraeOutput.midiDeviceOutput.WriteSysEx(bytes)
}
*/

/*
func (m *MIDIInput) Poll() (bool, error) {
	/*
		return m.stream.Poll()
	return false, nil
}
*/

/*
func (m *MIDIInput) ReadEvents() ([]MidiEvent, error) {
	events := make([]MidiEvent, 0)
	var err error
		// If you increase the value here,
		// be sure to actually handle all the events that come back
		rawEvents, err := m.stream.Read(1024)
		if err != nil {
			return nil, err
		}
		for n, e := range rawEvents {
			events[n] = MidiEvent{
				Timestamp: e.Timestamp,
				Status:    e.Status,
				Data1:     e.Data1,
				Data2:     e.Data2,
				SysEx:     e.SysEx,
				source:    m.Name(),
			}
		}
	return events, err
}
*/

func (mc *MIDIChannelOutput) SendBankProgram(bank int, program int) {
	if mc.bank != bank {
		Warn("SendBankProgram: XXX - SHOULD be sending", "bank", bank)
		mc.bank = bank
	}
	if mc.program != program {
		DebugLogOfType("midi", "SendBankProgram: sending", "program", program)
		mc.program = program
		status := byte(int64(ProgramStatus) | int64(mc.channel-1))
		data1 := byte(program - 1)
		mc.output.Send([]byte{status, data1})
	}
}

func (out *MIDIChannelOutput) WriteShort(status, data1, data2 int64) {
	Warn("WriteShort needs work")
}

func (m *MIDIIO) GetMidiChannelOutput(portchannel PortChannel) *MIDIChannelOutput {
	mc, ok := m.midiChannelOutputs[portchannel]
	if !ok {
		Warn("GetMidiChannelOutput: no entry", "port", portchannel.port, "channel", portchannel.channel)
		return nil
	}
	if mc.output == nil {
		Warn("GetMidiChannelOutput: midiDeviceOutput==nil", "port", portchannel.port, "channel", portchannel.channel)
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
		Warn("No output found matching", "port", portName)
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
