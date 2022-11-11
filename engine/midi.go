//go:build windows
// +build windows

package engine

import (
	"fmt"
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
		Log.Debugf("input port = %s\n", name)
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
		Log.Debugf("output port = %s\n", outp.String())
		if strings.Contains(name, "Erae Touch") {
			EraeOutput = outp
		}
		MIDI.midiOutputs[name] = outp
	}

	/*
		// for n := 0; n < ndevices; n++ {
			devid := portmidi.DeviceID(n)
			dev := portmidi.Info(devid)
			if dev.Name == "Erae Touch" {
				erae = true
			}
			if dev.IsOutputAvailable {
				if Debug.MIDI {
					Log.Debugf("MIDI OUTPUT device = %s  devid=%v\n", dev.Name, devid)
				}
				MIDI.outputDeviceID[dev.Name] = devid
				MIDI.outputDeviceInfo[dev.Name] = dev
				numOutputs++
			}
			if dev.IsInputAvailable {
				if Debug.MIDI {
					Log.Debugf("MIDI INPUT device = %s  devid=%v\n", dev.Name, devid)
				}
				MIDI.inputDeviceID[dev.Name] = devid
				MIDI.inputDeviceInfo[dev.Name] = dev
				numInputs++
			}
		}

		if erae {
			Log.Debugf("Erae Touch input is being enabled\n")
			InitErae()
		}

		Log.Debugf("MIDI devices (%d inputs, %d outputs) have been initialized\n", numInputs, numOutputs)
	*/

	if erae {
		Log.Debugf("Erae Touch input is being enabled\n")
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
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			fmt.Printf("starting note %s on channel %v with velocity %v\n", midi.Note(key), ch, vel)
		case msg.GetNoteEnd(&ch, &key):
			fmt.Printf("ending note %s on channel %v\n", midi.Note(key), ch)
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
		fmt.Printf("ERROR: %s\n", err)
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
		// Log.Debugf("\nmidiInput len(events)=%d\n", len(events))
	return events, err
}
*/

func (mc *MIDIChannelOutput) SendBankProgram(bank int, program int) {
	if mc.bank != bank {
		Log.Debugf("SendBankProgram: needs work\n")
		Log.Debugf("SendBankProgram: XXX - SHOULD be sending bank=%d\n", bank)
		mc.bank = bank
	}
	if mc.program != program {
		if Debug.MIDI {
			Log.Debugf("SendBankProgram: sending program=%d\n", program)
		}
		mc.program = program
		status := byte(int64(ProgramStatus) | int64(mc.channel-1))
		data1 := byte(program - 1)
		mc.output.Send([]byte{status, data1})
	}
}

// WriteSysEx sends one or more MIDI Events
/*
func (out *MIDIChannelOutput) WriteSysEx(bytes []byte) {
	Log.Debugf("WriteSysEx needs work\n")
		if out == nil {
			Log.Debugf("MIDIDeviceOutput.WriteSysEx: out is nil?\n")
			return
		}
		if Debug.MIDI {
			s := "["
			for _, b := range bytes {
				s += fmt.Sprintf(" 0x%02x", b)
			}
			s += " ]"
			Log.Debugf("WriteSysEx: bytes = %s\n", s)
		}
		if out.stream == nil {
			Log.Debugf("WriteSysEx: out.stream is nil?  port=%s\n", out.name)
			return
		}
		tm := time.Now()
		if err := out.stream.WriteSysExBytes(tm, bytes); err != nil {
			Log.Debugf("WriteSysExBytes: err=%s\n", err)
			return
		}
}
*/

func (out *MIDIChannelOutput) WriteShort(status, data1, data2 int64) {
	Log.Debugf("WriteShort needs work\n")
	/*
		if Debug.MIDI {
			Log.Debugf("MIDIDeviceOutput.WriteShort: status=0x%02x data1=%d data2=%d\n", status, data1, data2)
		}
		if out.stream == nil {
			Log.Debugf("SendEvent: out.stream is nil?  port=%s\n", out.name)
			return
		}
		if err := out.stream.WriteShort(status, data1, data2); err != nil {
			Log.Debugf("out.stream.WriteShort: err=%s\n", err)
			return
		}
	*/
}

/*
// GetOutputStream gets the Stream for a named port.  There can be multiple writers to an
// output stream; a cache of per-port output streams is kept in MIDIIO.outputDeviceStream
func (m *MIDIIO) getOutputStream(name string) (devid portmidi.DeviceID, stream *portmidi.Stream) {
	var present bool
	devid, present = m.outputDeviceID[name]
	if !present {
		Log.Debugf("getOutputStream: No such MIDI Output (%s)\n", name)
		return -1, nil
	}
	var err error
	stream, present = m.outputDeviceStream[name]
	if !present {
		Log.Debugf("Opening MIDI Output: %s\n", name)
		m.outputDeviceStream[name], err = portmidi.NewOutputStream(devid, 1, 0)
		if err != nil {
			Log.Debugf("getOutputStream: Unable to create NewOutputStream for %s\n", name)
			return -1, nil
		}
		stream = m.outputDeviceStream[name]
	}
	return devid, stream
}

// getInputStream gets the Stream for a named port
func (m *MIDIIO) getInputStream(name string) (devid portmidi.DeviceID, stream *portmidi.Stream) {
	var present bool
	devid, present = m.inputDeviceID[name]
	if !present {
		return -1, nil
	}
	var err error
	stream, present = m.inputDeviceStream[name]
	if !present {
		Log.Debugf("Opening MIDI Input: %s\n", name)
		m.inputDeviceStream[name], err = portmidi.NewInputStream(devid, 1024)
		if err != nil {
			Log.Debugf("portmidi.NewInputStream: err=%s\n", err)
			return -1, nil
		}
		stream = m.inputDeviceStream[name]
	}
	return devid, stream
}
*/

func (m *MIDIIO) GetMidiChannelOutput(portchannel PortChannel) *MIDIChannelOutput {
	mc, ok := m.midiChannelOutputs[portchannel]
	if !ok {
		Log.Debugf("GetMidiChannelOutput: no entry for port=%s channel=%d\n", portchannel.port, portchannel.channel)
		return nil
	}
	if mc.output == nil {
		Log.Debugf("GetMidiChannelOutput: midiDeviceOutput==nil for port=%s channel=%d\n", portchannel.port, portchannel.channel)
		return nil
	}
	if !mc.isopen {
		e := mc.output.Open()
		if e != nil {
			Log.Debugf("GetMidiChannelOutput: can't open %s - err=%s\n", mc.output.String(), e)
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
		Log.Debugf("No output found matching %s\n", portName)
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

/*
	devid, stream := m.getOutputStream(nm)
	if stream == nil {
		Log.Debugf("MIDIIO.openInput: Unable to open %s\n", nm)
		return nil
	}
	out := &MIDIDeviceOutput{name: nm, deviceID: devid, stream: stream}
	m.midiDeviceOutputs[nm] = out
	return out
*/

/*
func (m *MIDIIO) openInput(nm string) {
	Log.Debugf("MIDIIO.openInput: needs work\n")
		devid, stream := m.getInputStream(nm)
		if stream != nil {
			m.midiInputs[nm] = &MIDIInput{name: nm, deviceID: devid, stream: stream}
		} else {
			Log.Debugf("MIDIIO.openInput: Unable to open %s\n", nm)
		}
}
*/
