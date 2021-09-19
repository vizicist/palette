//go:build windows
// +build windows

package engine

import (
	"fmt"
	"log"

	"github.com/vizicist/portmidi"
)

// MIDIIO encapsulate everything having to do with MIDI I/O
type MIDIIO struct {
	// synth name is the key in these maps
	midiInputs  map[string]*MidiInput
	midiOutputs map[string]*MidiOutput
	// MIDI device name is the key in these maps
	outputDeviceID     map[string]portmidi.DeviceID
	outputDeviceInfo   map[string]*portmidi.DeviceInfo
	outputDeviceStream map[string]*portmidi.Stream
	inputDeviceID      map[string]portmidi.DeviceID
	inputDeviceInfo    map[string]*portmidi.DeviceInfo
	inputDeviceStream  map[string]*portmidi.Stream
}

type MidiInput struct {
	name     string
	deviceID portmidi.DeviceID
	stream   *portmidi.Stream
}

type MidiOutput struct {
	name     string
	deviceID portmidi.DeviceID
	stream   *portmidi.Stream
	// channel  int // 0-15 for MIDI channels 1-16
}

type Timestamp int64

// MidiEvent is very similar to portmidi.Event, but I don't want to tie things
// too directly to portmidi, so that it can be replaced easily if necessary.
type MidiEvent struct {
	Timestamp Timestamp
	Status    int64 // if == 0xF0, use SysEx, otherwise Data1 and Data2.
	Data1     int64
	Data2     int64
	SysEx     []byte
	source    string // {midiinputname} or maybe {IPaddress}.{midiinputname}
}

func (out MidiOutput) Name() string {
	return out.name
}

func (in MidiInput) Name() string {
	return in.name
}

// MIDI is a pointer to
var MIDI *MIDIIO

// InitMIDI initializes stuff
func InitMIDI() {

	InitializeClicksPerSecond(defaultClicksPerSecond)

	MIDI = &MIDIIO{
		midiInputs:         make(map[string]*MidiInput),
		midiOutputs:        make(map[string]*MidiOutput),
		outputDeviceID:     make(map[string]portmidi.DeviceID),
		outputDeviceInfo:   make(map[string]*portmidi.DeviceInfo),
		outputDeviceStream: make(map[string]*portmidi.Stream),
		inputDeviceID:      make(map[string]portmidi.DeviceID),
		inputDeviceInfo:    make(map[string]*portmidi.DeviceInfo),
		inputDeviceStream:  make(map[string]*portmidi.Stream),
	}

	// util.InitScales()

	portmidi.Initialize()

	ndevices := portmidi.CountDevices()
	numInputs := 0
	numOutputs := 0
	erae := false
	for n := 0; n < ndevices; n++ {
		devid := portmidi.DeviceID(n)
		dev := portmidi.Info(devid)
		if dev.Name == "Erae Touch" {
			erae = true
		}
		if dev.IsOutputAvailable {
			// log.Printf("MIDI OUTPUT device = %s  devid=%v\n", dev.Name, devid)
			MIDI.outputDeviceID[dev.Name] = devid
			MIDI.outputDeviceInfo[dev.Name] = dev
			numOutputs++
		}
		if dev.IsInputAvailable {
			// log.Printf("MIDI INPUT device = %s  devid=%v\n", dev.Name, devid)
			MIDI.inputDeviceID[dev.Name] = devid
			MIDI.inputDeviceInfo[dev.Name] = dev
			numInputs++
		}
	}

	midiinput := ConfigValue("midiinput")
	if midiinput != "" {
		MIDI.openInput(midiinput)
	}

	if erae {
		log.Printf("Erae Touch input is being enabled\n")
		InitErae()
	}

	log.Printf("MIDI devices (%d inputs, %d outputs) have been initialized\n", numInputs, numOutputs)
}

func (m *MidiInput) Poll() (bool, error) {
	return m.stream.Poll()
}

func (m *MidiInput) ReadEvents() ([]MidiEvent, error) {
	// If you increase the value here,
	// be sure to actually handle all the events that come back
	rawEvents, err := m.stream.Read(1024)
	if err != nil {
		return nil, err
	}
	events := make([]MidiEvent, len(rawEvents))
	for n, e := range rawEvents {
		events[n] = MidiEvent{
			Timestamp: Timestamp(e.Timestamp),
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

/*
func (m *MIDIIO) GetMidiOutput(synth string) *midiOutput {
	s, ok := m.midiOutputs[synth]
	if !ok {
		s, ok = m.midiOutputs[defaultSynth]
		if !ok {
			return nil
		}
	}
	return s
}
*/

// WriteSysex sends one or more MIDI Events
func (out *MidiOutput) WriteSysex(bytes []byte) {
	if out == nil {
		log.Printf("MidiOutput.WriteSysex: out is nil?\n")
		return
	}
	if DebugUtil.MIDI {
		s := "["
		for _, b := range bytes {
			s += fmt.Sprintf(" 0x%02x", b)
		}
		s += " ]"
		log.Printf("WriteSysex: bytes = %s\n", s)
	}
	if out.stream == nil {
		log.Printf("WriteSysex: out.stream is nil?  port=%s\n", out.name)
		return
	}
	tm := portmidi.Time()
	if err := out.stream.WriteSysExBytes(tm, bytes); err != nil {
		log.Printf("WriteSysExBytes: err=%s\n", err)
		return
	}
}

func (out *MidiOutput) WriteShort(status, data1, data2 int64) {
	if DebugUtil.MIDI {
		log.Printf("MidiOutput.WriteShort: status=0x%02x data1=%d data2=%d\n", status, data1, data2)
	}
	if out.stream == nil {
		log.Printf("SendEvent: out.stream is nil?  port=%s\n", out.name)
		return
	}
	if err := out.stream.WriteShort(status, data1, data2); err != nil {
		log.Printf("out.stream.WriteShort: err=%s\n", err)
		return
	}
}

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

func (m *MIDIIO) GetMidiOutput(name string) *MidiOutput {
	midiout, ok := m.midiOutputs[name]
	if !ok {
		return nil
	}
	return midiout
}

func (m *MIDIIO) GetMidiInput(name string) *MidiInput {
	midiin, ok := m.midiInputs[name]
	if !ok {
		return nil
	}
	return midiin
}

func (m *MIDIIO) openOutput(nm string) *MidiOutput {
	devid, stream := m.getOutputStream(nm)
	if stream == nil {
		log.Printf("MIDIIO.openInput: Unable to open %s\n", nm)
		return nil
	}
	out := &MidiOutput{name: nm, deviceID: devid, stream: stream}
	m.midiOutputs[nm] = out
	return out
}

func (m *MIDIIO) openFakeOutput(port string) *MidiOutput {
	return &MidiOutput{name: port, deviceID: -1, stream: nil}
}

func (m *MIDIIO) openInput(nm string) {
	devid, stream := m.getInputStream(nm)
	if stream != nil {
		m.midiInputs[nm] = &MidiInput{name: nm, deviceID: devid, stream: stream}
	} else {
		log.Printf("MIDIIO.openInput: Unable to open %s\n", nm)
	}
}

func (m *MIDIIO) InputMap() map[string]*MidiInput {
	return m.midiInputs
}
