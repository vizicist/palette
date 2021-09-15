//go:build windows
// +build windows

package engine

import (
	"fmt"
	"log"
	"strings"

	"github.com/vizicist/portmidi"
)

// MIDIIO encapsulate everything having to do with MIDI I/O
type MIDIIO struct {
	// synth name is the key in these maps
	midiInputs map[string]*MidiInput
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

func (in MidiInput) Name() string {
	return in.name
}

type MidiOutput struct {
	name     string
	deviceID portmidi.DeviceID
	stream   *portmidi.Stream
	channel  int // 0-15 for MIDI channels 1-16
}

func (out MidiOutput) Name() string {
	return out.name
}

// MIDI is a pointer to
var MIDI *MIDIIO

// InitMIDI initializes stuff
func InitMIDI(midiinput string) {

	InitializeClicksPerSecond(defaultClicksPerSecond)

	m := &MIDIIO{
		midiInputs:         make(map[string]*MidiInput),
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
	for n := 0; n < ndevices; n++ {
		devid := portmidi.DeviceID(n)
		dev := portmidi.Info(devid)
		if dev.IsOutputAvailable {
			// log.Printf("MIDI OUTPUT device = %s  devid=%v\n", dev.Name, devid)
			m.outputDeviceID[dev.Name] = devid
			m.outputDeviceInfo[dev.Name] = dev
			numOutputs++
		}
		if dev.IsInputAvailable {
			// log.Printf("MIDI INPUT device = %s  devid=%v\n", dev.Name, devid)
			m.inputDeviceID[dev.Name] = devid
			m.inputDeviceInfo[dev.Name] = dev
			numInputs++
		}
	}
	if midiinput != "" {
		m.loadInputs(midiinput)
	}
	MIDI = m
	log.Printf("MIDI devices (%d inputs, %d outputs) have been initialized\n", numInputs, numOutputs)
}

func (m *MidiInput) Poll() (bool, error) {
	return m.stream.Poll()
}

func (m *MidiInput) ReadEvents() ([]portmidi.Event, error) {
	// If you increase the value here,
	// be sure to actually handle all the events that come back
	events, err := m.stream.Read(1024)
	if err != nil {
		return nil, err
	}
	// log.Printf("\nmidiInput len(events)=%d\n", len(events))
	return events, err
}

/*
func (m *MIDIIO) NewMidiOutput(synth string) *midiOutput {
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

// SendEvent sends one or more MIDI Events
func (out *MidiOutput) WriteSysex(bytes []byte) {
	if DebugUtil.MIDI {
		s := "["
		for _, b := range bytes {
			s += fmt.Sprintf(" 0x%02x", b)
		}
		s += " ]\n"
		log.Printf("SendSysex: bytes = %s\n", s)
	}
	if out.stream == nil {
		log.Printf("SendEvent: out.stream is nil?  port=%s\n", out.name)
		return
	}
	tm := portmidi.Time()
	if err := out.stream.WriteSysExBytes(tm, bytes); err != nil {
		log.Printf("out.stream.Write: err=%s\n", err)
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
		m.outputDeviceStream[name], err = portmidi.NewOutputStream(devid, 1, 0)
		if err != nil {
			log.Printf("getOutputStream: Unable to create NewOutputStream for %s\n", name)
			return -1, nil
		}
		stream = m.outputDeviceStream[name]
	}
	return devid, stream
}

// GetInputStream gets the Stream for a named port
func (m *MIDIIO) getInputStream(name string) (devid portmidi.DeviceID, stream *portmidi.Stream) {
	var present bool
	devid, present = m.inputDeviceID[name]
	if !present {
		return -1, nil
	}
	var err error
	stream, present = m.inputDeviceStream[name]
	if !present {
		m.inputDeviceStream[name], err = portmidi.NewInputStream(devid, 1024)
		if err != nil {
			log.Printf("portmidi.NewInputStream: err=%s\n", err)
			return -1, nil
		}
		stream = m.inputDeviceStream[name]
	}
	return devid, stream
}

func (m *MIDIIO) NewMidiOutput(name string, channel int) *MidiOutput {
	devid, stream := m.getOutputStream(name)
	if stream == nil {
		log.Printf("NewMidiOutput: failed to get OutpuStream for name=%s\n", name)
		return nil
	}
	return &MidiOutput{name: name, deviceID: devid, stream: stream, channel: channel}
}

func (m *MIDIIO) NewMidiInput(name string) *MidiInput {
	devid, stream := m.getInputStream(name)
	if stream == nil {
		log.Printf("NewMidiOutput: failed to get OutpuStream for name=%s\n", name)
		return nil
	}
	return &MidiInput{name: name, deviceID: devid, stream: stream}
}

func (m *MIDIIO) NewFakeMidiOutput(port string, channel int) *MidiOutput {
	return &MidiOutput{name: port, deviceID: -1, stream: nil, channel: channel}
}

func (m *MIDIIO) loadInputs(dev string) {
	// Should load this from settings.json file
	words := strings.Split(dev, ",")
	for _, nm := range words {
		devid, stream := m.getInputStream(nm)
		if stream != nil {
			log.Printf("MIDIIO.loadInputs: Successfully opened %s\n", nm)
			m.midiInputs[nm] = &MidiInput{deviceID: devid, stream: stream}
		} else {
			log.Printf("MIDIIO.loadInputs: Unable to open %s\n", nm)
		}
	}
}

func (m *MIDIIO) InputMap() map[string]*MidiInput {
	return m.midiInputs
}
