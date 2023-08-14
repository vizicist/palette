//go:build windows
// +build windows

package hostwin

import (
	"fmt"
	"strings"

	// "sync"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
	"github.com/vizicist/palette/kit"
)

// MidiIO encapsulate everything having to do with MIDI I/O
type MidiIO struct {
	midiInput            drivers.In
	midiOutputs          map[string]drivers.Out // synth name is the key
	midiPortChannelState map[kit.PortChannel]*kit.MIDIPortChannelState
	inports              midi.InPorts
	outports             midi.OutPorts
	stop                 func()
}

func DefaultMIDIChannelOutput() *kit.MIDIPortChannelState {
	return &kit.MIDIPortChannelState{}
}

// MIDI is a pointer to
var TheMidiIO *MidiIO

func NewMidiIO() *MidiIO {

	midiIO := &MidiIO{
		midiInput:            nil,
		midiOutputs:          make(map[string]drivers.Out),
		midiPortChannelState: make(map[kit.PortChannel]*kit.MIDIPortChannelState),
		stop:                 nil,
		inports:              midi.GetInPorts(),
		outports:             midi.GetOutPorts(),
	}

	for _, outp := range midiIO.outports {
		name := outp.String()
		// NOTE: name is the port name followed by an index
		LogOfType("midiports", "MIDI output", "port", outp.String())
		if strings.Contains(name, "Erae Touch") {
			kit.TheErae.Output = outp
		}
		midiIO.midiOutputs[name] = outp
	}

	LogInfo("Initialized MIDI", "numoutports", len(midiIO.outports))
	LogOfType("midi", "MIDI outports", "outports", midiIO.outports)

	// if erae {
	// 	Info("Erae Touch input is being enabled")
	// 	InitErae()
	// }

	return midiIO
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
	TheRouter.midiInputChan <- kit.NewMidiEvent(kit.CurrentClick(), "handleMidiInput", msg)

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

// func (out *kit.MIDIPortChannelState) WriteShort(status, data1, data2 int64) {
// 	LogWarn("WriteShort needs work")
// }

func (h HostWin) GetPortChannelState(portchannel kit.PortChannel) (*kit.MIDIPortChannelState, error) {

	state, ok := TheMidiIO.midiPortChannelState[portchannel]
	if !ok {
		return nil, fmt.Errorf("GetMidiChannelOutput: no entry, port=%s channel=%d", portchannel.Port, portchannel.Channel)
	}
	if state.Output == nil {
		return nil, fmt.Errorf("GetMidiChannelOutput: midiDeviceOutput==nil, port=%s channel=%d", portchannel.Port, portchannel.Channel)
	}

	state.Mutex.Lock()
	defer state.Mutex.Unlock()
	if !state.Isopen {
		e := state.Output.Open()
		if e != nil {
			return nil, e
		}
	}
	state.Isopen = true
	return state, nil
}

func (h HostWin) OpenChannelOutput(portchannel kit.PortChannel) *kit.MIDIPortChannelState {
	return TheMidiIO.openChannelOutput(portchannel)
}

func (m *MidiIO) openChannelOutput(portchannel kit.PortChannel) *kit.MIDIPortChannelState {

	portName := portchannel.Port
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

	state := &kit.MIDIPortChannelState{
		Channel: portchannel.Channel,
		Bank:    0,
		Program: 0,
		Output:  output,
		PortName: portName,
		Isopen:  false,
	}
	m.midiPortChannelState[portchannel] = state
	return state
}

func (h HostWin) OpenFakeChannelOutput(port string, channel int) *kit.MIDIPortChannelState {
	return TheMidiIO.openFakeChannelOutput(port,channel)
}

func (m *MidiIO) openFakeChannelOutput(port string, channel int) *kit.MIDIPortChannelState {

	co := &kit.MIDIPortChannelState{
		Channel: channel,
		Bank:    0,
		Program: 0,
		Output:  nil,
		PortName: "fake",
		Isopen:  false,
	}
	return co
}

