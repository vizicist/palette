//go:build windows
// +build windows

package engine

import (
	"fmt"
	"log"
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
	channel          int
	bank             int
	program          int
	midiDeviceOutput drivers.Out
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
		log.Printf("input port = %s\n", name)
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
		log.Printf("output port = %s\n", outp.String())
		if name == "Erae Touch" {
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
					log.Printf("MIDI OUTPUT device = %s  devid=%v\n", dev.Name, devid)
				}
				MIDI.outputDeviceID[dev.Name] = devid
				MIDI.outputDeviceInfo[dev.Name] = dev
				numOutputs++
			}
			if dev.IsInputAvailable {
				if Debug.MIDI {
					log.Printf("MIDI INPUT device = %s  devid=%v\n", dev.Name, devid)
				}
				MIDI.inputDeviceID[dev.Name] = devid
				MIDI.inputDeviceInfo[dev.Name] = dev
				numInputs++
			}
		}

		if erae {
			log.Printf("Erae Touch input is being enabled\n")
			InitErae()
		}

		log.Printf("MIDI devices (%d inputs, %d outputs) have been initialized\n", numInputs, numOutputs)
	*/

	if erae {
		log.Printf("Erae Touch input is being enabled\n")
		InitErae()
	}
}

func (m *MIDIIO) Start() {
	if MIDI.Input == nil {
		log.Printf("MIDIIO.Start: no Midi input\n")
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
		// log.Printf("\nmidiInput len(events)=%d\n", len(events))
	return events, err
}
*/

func (out *MIDIChannelOutput) SendBankProgram(bank int, program int) {
	log.Printf("SendBankProgram needs work\n")
	/*
		if out.bank != bank {
			log.Printf("SendBankProgram: XXX - SHOULD be sending bank=%d\n", bank)
			out.bank = bank
		}
		if out.program != program {
			if Debug.MIDI {
				log.Printf("SendBankProgram: sending program=%d\n", program)
			}
			out.program = program
			status := int64(ProgramStatus) | int64(out.channel-1)
			data1 := int64(program - 1)
			out.midiDeviceOutput.stream.WriteShort(status, data1, 0)
		}
	*/
}

// WriteSysEx sends one or more MIDI Events
/*
func (out *MIDIChannelOutput) WriteSysEx(bytes []byte) {
	log.Printf("WriteSysEx needs work\n")
		if out == nil {
			log.Printf("MIDIDeviceOutput.WriteSysEx: out is nil?\n")
			return
		}
		if Debug.MIDI {
			s := "["
			for _, b := range bytes {
				s += fmt.Sprintf(" 0x%02x", b)
			}
			s += " ]"
			log.Printf("WriteSysEx: bytes = %s\n", s)
		}
		if out.stream == nil {
			log.Printf("WriteSysEx: out.stream is nil?  port=%s\n", out.name)
			return
		}
		tm := time.Now()
		if err := out.stream.WriteSysExBytes(tm, bytes); err != nil {
			log.Printf("WriteSysExBytes: err=%s\n", err)
			return
		}
}
*/

func (out *MIDIChannelOutput) WriteShort(status, data1, data2 int64) {
	log.Printf("WriteShort needs work\n")
	/*
		if Debug.MIDI {
			log.Printf("MIDIDeviceOutput.WriteShort: status=0x%02x data1=%d data2=%d\n", status, data1, data2)
		}
		if out.stream == nil {
			log.Printf("SendEvent: out.stream is nil?  port=%s\n", out.name)
			return
		}
		if err := out.stream.WriteShort(status, data1, data2); err != nil {
			log.Printf("out.stream.WriteShort: err=%s\n", err)
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
		log.Printf("getOutputStream: No such MIDI Output (%s)\n", name)
		return -1, nil
	}
	var err error
	stream, present = m.outputDeviceStream[name]
	if !present {
		log.Printf("Opening MIDI Output: %s\n", name)
		m.outputDeviceStream[name], err = portmidi.NewOutputStream(devid, 1, 0)
		if err != nil {
			log.Printf("getOutputStream: Unable to create NewOutputStream for %s\n", name)
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
		log.Printf("Opening MIDI Input: %s\n", name)
		m.inputDeviceStream[name], err = portmidi.NewInputStream(devid, 1024)
		if err != nil {
			log.Printf("portmidi.NewInputStream: err=%s\n", err)
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
		log.Printf("GetMidiChannelOutput: no entry for port=%s channel=%d\n", portchannel.port, portchannel.channel)
		return nil
	}
	if mc.midiDeviceOutput == nil {
		log.Printf("GetMidiChannelOutput: midiDeviceOutput==nil for port=%s channel=%d\n", portchannel.port, portchannel.channel)
		return nil
	}
	log.Printf("GetMidiChannelOutput needs work\n")
	/*
		if mc.midiDeviceOutput.stream == nil {
			log.Printf("GetMidiChannelOutput: midiDeviceOutput.stream==nil for port=%s channel=%d\n", portchannel.port, portchannel.channel)
			return nil
		}
	*/
	return mc
}

func (m *MIDIIO) openChannelOutput(portchannel PortChannel) *MIDIChannelOutput {

	output := m.midiOutputs[portchannel.port]

	mc := &MIDIChannelOutput{
		channel:          portchannel.channel,
		bank:             0,
		program:          0,
		midiDeviceOutput: output,
	}
	m.midiChannelOutputs[portchannel] = mc
	return mc
}
func (m *MIDIIO) openFakeChannelOutput(port string, channel int) *MIDIChannelOutput {

	co := &MIDIChannelOutput{
		channel:          channel,
		bank:             0,
		program:          0,
		midiDeviceOutput: nil,
	}
	return co
}

/*
	devid, stream := m.getOutputStream(nm)
	if stream == nil {
		log.Printf("MIDIIO.openInput: Unable to open %s\n", nm)
		return nil
	}
	out := &MIDIDeviceOutput{name: nm, deviceID: devid, stream: stream}
	m.midiDeviceOutputs[nm] = out
	return out
*/

/*
func (m *MIDIIO) openInput(nm string) {
	log.Printf("MIDIIO.openInput: needs work\n")
		devid, stream := m.getInputStream(nm)
		if stream != nil {
			m.midiInputs[nm] = &MIDIInput{name: nm, deviceID: devid, stream: stream}
		} else {
			log.Printf("MIDIIO.openInput: Unable to open %s\n", nm)
		}
}
*/
