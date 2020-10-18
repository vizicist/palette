// +build windows

package palette

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/vizicist/portmidi"
)

// MIDIIO encapsulate everything having to do with MIDI I/O
type MIDIIO struct {
	// synth name is the key in these maps
	synthOutputs map[string]*synthOutput
	midiInputs   map[string]*midiInput
	// MIDI device name is the key in these maps
	outputDeviceID     map[string]portmidi.DeviceID
	outputDeviceInfo   map[string]*portmidi.DeviceInfo
	outputDeviceStream map[string]*portmidi.Stream
	inputDeviceID      map[string]portmidi.DeviceID
	inputDeviceInfo    map[string]*portmidi.DeviceInfo
	inputDeviceStream  map[string]*portmidi.Stream
}

type midiInput struct {
	name     string
	deviceID portmidi.DeviceID
	stream   *portmidi.Stream
}

type synthOutput struct {
	port     string
	deviceID portmidi.DeviceID
	stream   *portmidi.Stream
	channel  int // 0-15 for MIDI channels 1-16
}

// MIDIEvent is a single MIDI input event
// type MIDIEvent struct {
// 	portmidi.Event
// }

// MIDI is a pointer to
var MIDI *MIDIIO

// InitMIDI initializes stuff
func InitMIDI() {

	InitializeClicksPerSecond(defaultClicksPerSecond)

	m := &MIDIIO{
		synthOutputs:       make(map[string]*synthOutput),
		midiInputs:         make(map[string]*midiInput),
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
	m.loadSynths(configFile("Synths.json"))
	midiInput := ConfigValue("midiinput")
	if midiInput != "" {
		m.loadInputs(midiInput)
	}
	MIDI = m
	log.Printf("MIDI devices (%d inputs, %d outputs) have been initialized\n", numInputs, numOutputs)
}

func (m *midiInput) Poll() (bool, error) {
	return m.stream.Poll()
}

func (m *midiInput) ReadEvent() (portmidi.Event, error) {
	// If you increase the value here,
	// be sure to actually handle all the events that come back
	events, err := m.stream.Read(1)
	if err != nil {
		return portmidi.Event{}, err
	}
	// log.Printf("\nmidiInput len(events)=%d\n", len(events))
	me := events[0]
	return me, err
}

func (m *MIDIIO) getOutput(synth string) *synthOutput {
	s, ok := m.synthOutputs[synth]
	if !ok {
		s, ok = m.synthOutputs[defaultSynth]
		if !ok {
			return nil
		}
	}
	return s
}

func (m *MIDIIO) getInput(dev string) *midiInput {
	s, ok := m.midiInputs[dev]
	if !ok {
		return nil
	}
	return s
}

// SendANO sends all-notes-off
func (m *MIDIIO) SendANO(synth string) {
	s := m.getOutput(synth)
	if s == nil {
		log.Printf("Hey, SendANO finds no SynthOutput for %s\n", synth)
		return
	}
	status := 0xb0 | (s.channel - 1)
	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(status),
		Data1:     int64(0x7b),
		Data2:     int64(0x00),
	}
	// log.Printf("Sending ANO portmidi.Event = %s\n", e)
	SendEvent(s, []portmidi.Event{e})
}

// SendNote sends MIDI output for a Note
func (m *MIDIIO) SendNote(n *Note) {
	s := m.getOutput(n.Sound)
	if s == nil {
		log.Printf("Hey, SendNote finds no SynthOutput for %s\n", n.Sound)
		return
	}

	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(s.channel - 1), // pre-populate with the channel
		Data1:     int64(n.Pitch),
		Data2:     int64(n.Velocity),
	}
	switch n.TypeOf {
	case NOTEON:
		e.Status |= 0x90
	case NOTEOFF:
		e.Status |= 0x80
	case CONTROLLER:
		e.Status |= 0xB0
	case PROGCHANGE:
		e.Status |= 0xC0
	case CHANPRESSURE:
		e.Status |= 0xD0
	case PITCHBEND:
		e.Status |= 0xE0
	default:
		log.Printf("SendNote can't handle Note TypeOf=%v\n", n.TypeOf)
		return
	}

	// log.Printf("Sending portmidi.Event = %s\n", e)
	SendEvent(s, []portmidi.Event{e})
}

// Send writes to a MIDI output
func Send(out *synthOutput, b1 int64, b2 int64, b3 int64) {
	if err := out.stream.WriteShort(b1, b2, b3); err != nil {
		log.Fatal(err)
	}
}

// SendEvent sends one or more MIDI Events
func SendEvent(out *synthOutput, events []portmidi.Event) {
	if out.stream == nil {
		log.Printf("SendEvent: out.stream is nil?  port=%s\n", out.port)
		return
	}
	if err := out.stream.Write(events); err != nil {
		log.Fatal(err)
	}
}

// GetOutputStream gets the Stream for a named port
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
			log.Fatal(err)
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
		m.inputDeviceStream[name], err = portmidi.NewInputStream(devid, 128)
		if err != nil {
			log.Printf("portmidi.NewInputStream: err=%s\n", err)
			return -1, nil
		}
		stream = m.inputDeviceStream[name]
	}
	return devid, stream
}

func (m *MIDIIO) makeSynthOutput(port string, channel int) *synthOutput {
	devid, stream := m.getOutputStream(port)
	return &synthOutput{port: port, deviceID: devid, stream: stream, channel: channel}
}

func (m *MIDIIO) loadSynths(filename string) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	var synths synths
	json.Unmarshal(bytes, &synths)

	for i := range synths.Synths {
		nm := synths.Synths[i].Name
		port := synths.Synths[i].Port
		channel := synths.Synths[i].Channel
		m.synthOutputs[nm] = m.makeSynthOutput(port, channel)
	}
}

func (m *MIDIIO) loadInputs(dev string) {
	// Should load this from settings.json file
	devid, stream := m.getInputStream(dev)
	if stream != nil {
		m.midiInputs[dev] = &midiInput{deviceID: devid, stream: stream}
	}
}
