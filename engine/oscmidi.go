// +build osc

package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/vizicist/portmidi"
)

// OscmidiDevice encapsulate everything having to do with MIDI I/O
type OscmidiDevice struct {
	// synth name is the key in these maps
	soundOutputs    map[string]*soundOutput
	MidiInputs      map[string]*midiInput
	MidiInputNames  []string
	listeningNames  []string
	MidiOutputNames []string
	// MIDI device name is the key in these maps
	// outputDeviceID     map[string]portmidi.DeviceID
	// outputDeviceInfo   map[string]*portmidi.DeviceInfo
	// outputDeviceStream map[string]*portmidi.Stream
	// inputDeviceID      map[string]portmidi.DeviceID
	// inputDeviceInfo    map[string]*portmidi.DeviceInfo
	// inputDeviceStream  map[string]*portmidi.Stream
}

type midiInput struct {
	name string
	// deviceID portmidi.DeviceID
	// stream   *portmidi.Stream
}

type soundOutput struct {
	// port string
	// deviceID portmidi.DeviceID
	// stream   *portmidi.Stream
	// 1-based MIDI channel, a value of 0 means unspecified, though
	// though I'm not sure unspecified should be allowed,
	// and it may be disallowed at some point.
	channel uint8
	program uint8
}

// PortmidiEvent is a single MIDI input event
// XXX - since is local, PortmidiEvent should just be portmidi.Event
// type PortmidiEvent struct {
// 	Event *portmidi.Event
// }

// NewOscmidiDevice returns a MidiDevice interface for OSC to Element
func NewOscmidiDevice() (MidiDevice, error) {

	pm := &OscmidiDevice{
		soundOutputs:    make(map[string]*soundOutput),
		MidiInputs:      make(map[string]*midiInput),
		MidiInputNames:  make([]string, 0),
		listeningNames:  make([]string, 0),
		MidiOutputNames: make([]string, 0),
		// outputDeviceID:     make(map[string]portmidi.DeviceID),
		// outputDeviceInfo:   make(map[string]*portmidi.DeviceInfo),
		// outputDeviceStream: make(map[string]*portmidi.Stream),
		// inputDeviceID:      make(map[string]portmidi.DeviceID),
		// inputDeviceInfo:    make(map[string]*portmidi.DeviceInfo),
		// inputDeviceStream:  make(map[string]*portmidi.Stream),
	}
	return pm, nil

	/*
		portmidi.Initialize()

		ndevices := portmidi.CountDevices()
		numInputs := 0
		numOutputs := 0
		for n := 0; n < ndevices; n++ {
			devid := portmidi.DeviceID(n)
			dev := portmidi.Info(devid)
			if dev.IsOutputAvailable {
				// log.Printf("MIDI OUTPUT device = %s  devid=%v\n", dev.Name, devid)
				pm.outputDeviceID[dev.Name] = devid
				pm.outputDeviceInfo[dev.Name] = dev
				pm.MidiOutputNames = append(pm.MidiOutputNames, dev.Name)
				numOutputs++
			}
			if dev.IsInputAvailable {
				// log.Printf("MIDI INPUT device = %s  devid=%v\n", dev.Name, devid)
				pm.inputDeviceID[dev.Name] = devid
				pm.inputDeviceInfo[dev.Name] = dev
				pm.MidiInputNames = append(pm.MidiInputNames, dev.Name)
				numInputs++
			}
		}

		log.Printf("PortMIDI (%d inputs, %d outputs) has been initialized\n", numInputs, numOutputs)
	*/
}

// ReadEvent xxx
func (m OscmidiDevice) ReadEvent(inputName string) (MidiDeviceEvent, error) {
	input := m.MidiInputs[inputName]
	pmEvent, err := input.ReadEvent()
	if err != nil {
		return MidiDeviceEvent{}, fmt.Errorf("OscmidiDevice.ReadEvent: err=%s", err)
	}
	e := MidiDeviceEvent{
		Timestamp: int64(pmEvent.Event.Timestamp),
		Status:    pmEvent.Event.Status,
		Data1:     pmEvent.Event.Data1,
		Data2:     pmEvent.Event.Data2,
	}
	return e, nil
}

// Poll xxx
func (m OscmidiDevice) Poll(inputName string) (bool, error) {
	// stream := m.inputDeviceStream[inputName]
	// return stream.Poll()
	return false, nil
}

// func (m *midiInput) Poll() (bool, error) {
// 	return m.stream.Poll()
// }

func (m *midiInput) ReadEvent() (PortmidiEvent, error) {
	return PortmidiEvent{}, fmt.Errorf("oscmidi.ReadEvent does nothing")
	/*
		events, err := m.stream.Read(1024)
		if err != nil {
			return PortmidiEvent{}, err
		}
		// log.Printf("midiInput len(events)=%d\n", len(events))
		me := PortmidiEvent{Event: &events[0]}
		return me, err
	*/
}

func (m *OscmidiDevice) getSoundOutput(synth string) (*soundOutput, error) {
	if synth == "" {
		synth = "default"
	}
	s, ok := m.soundOutputs[synth]
	if !ok {
		return nil, fmt.Errorf("OscmidiDevice.getSoundOutput can't find output for synth=\"%s\"", synth)
	}
	// Sanity check on the channel
	if s.channel == 0 {
		return nil, fmt.Errorf("OscmidiDevice.getSoundOutput finds 0 channel value for synth=\"%s\"", synth)
	}
	return s, nil
}

/*
func (m *OscmidiDevice) getInput(dev string) *midiInput {
	s, ok := m.midiInputs[dev]
	if !ok {
		return nil
	}
	return s
}
*/

// SendANO sends all-notes-off
func (m *OscmidiDevice) SendANO(synth string) {
	s, err := m.getSoundOutput(synth)
	if err != nil {
		log.Printf("OscmidiDevice.SendANO error: %s\n", err)
		return
	}
	if s == nil {
		if synth != "" {
			log.Printf("Hey, OscmidiDevice.SendANO finds no SoundOutput for %s\n", synth)
		}
		return
	}
	// NOTE: s.channel is 1-based, but MIDI output is 0-based
	status := 0xb0 | (s.channel - 1)
	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(status),
		Data1:     int64(0x7b),
		Data2:     int64(0x00),
	}
	oscsendEvent(s, []portmidi.Event{e})
}

// Inputs returns a list of MIDI input names
func (m OscmidiDevice) Inputs() []string {
	return m.MidiInputNames
}

// ListeningInputs returns a list of MIDI input names that are open for listening
func (m *OscmidiDevice) ListeningInputs() []string {
	return m.listeningNames
}

// Outputs returns a list of MIDI output names
func (m *OscmidiDevice) Outputs() []string {
	return m.MidiOutputNames
}

// SendNote sends MIDI output for a Note
func (m *OscmidiDevice) SendNote(n *Note, debug bool, callbacks []*NoteOutputCallback) {

	// XXX - probably want to do someting about looking up the SoundOutput every time
	// XXX - perhaps store it in ActiveNote

	s, err := m.getSoundOutput(n.Sound)
	if err != nil {
		log.Printf("OscmidiDevice.SendNote error: %s\n", err)
		return
	}
	var status uint8
	switch n.TypeOf {
	case NOTEON:
		status = 0x90
	case NOTEOFF:
		status = 0x80
	default:
		log.Printf("SendNote can't YET handle Note TypeOf=%v\n", n.TypeOf)
		return
	}
	// NOTE: s.channel is 1-based, but MIDI output is 0-based
	status |= (s.channel - 1)
	e := portmidi.Event{
		Timestamp: portmidi.Time(),
		Status:    int64(status),
		Data1:     int64(n.Pitch),
		Data2:     int64(n.Velocity),
	}
	if debug {
		log.Printf("MIDI.SendNote status=0x%0x pitch=%d velocity=%d\n", status, n.Pitch, n.Velocity)
	}
	// Other than ANO events, this is the one and only place
	// where things are sent to the MIDI output
	for _, cb := range callbacks {
		cb.Callback(n)
	}

	oscsendEvent(s, []portmidi.Event{e})
}

// oscsendEvent sends one or more MIDI Events
func oscsendEvent(out *soundOutput, events []portmidi.Event) {
	log.Printf("oscmidi.oscsendEvent does nothing!\n")
	/*
		if err := out.stream.Write(events); err != nil {
			log.Printf("oscmidi.oscsendEvent: err=%s\n", err)
		}
	*/
}

/*
// GetOutputStream gets the Stream for a named port
func (m *OscmidiDevice) getOutputStream(name string) (devid portmidi.DeviceID, stream *portmidi.Stream, err error) {
	var present bool
	devid, present = m.outputDeviceID[name]
	if !present {
		return -1, nil, fmt.Errorf(fmt.Sprintf("oscmidi.getOutputStream - No such MIDI Output (%s)\n", name))
	}
	stream, present = m.outputDeviceStream[name]
	if !present {
		m.outputDeviceStream[name], err = portmidi.NewOutputStream(devid, 1, 0)
		if err != nil {
			return -1, nil, fmt.Errorf("portmidi.NewOutputStream: err=%s", err)
		}
		stream = m.outputDeviceStream[name]
	}
	return devid, stream, nil
}
*/

/*
// GetInputStream gets the Stream for a named port
func (m *OscmidiDevice) getInputStream(name string) (devid portmidi.DeviceID, stream *portmidi.Stream, err error) {
	var present bool
	devid, present = m.inputDeviceID[name]
	if !present {
		return -1, nil, fmt.Errorf("no such MIDI input - %s", name)
	}
	stream, present = m.inputDeviceStream[name]
	if !present {
		m.inputDeviceStream[name], err = portmidi.NewInputStream(devid, 32)
		if err != nil {
			return -1, nil, err
		}
		stream = m.inputDeviceStream[name]
	}
	return devid, stream, nil
}
*/

func (m *OscmidiDevice) makeSoundOutput(channel uint8, program uint8) (*soundOutput, error) {
	return &soundOutput{channel: channel, program: program}, nil
}

// LoadSounds xxx
func (m *OscmidiDevice) LoadSounds(filename string) error {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return err
	}

	synths := f.([]interface{})
	for _, dat := range synths {
		smap := dat.(map[string]interface{})
		name := smap["name"].(string)
		port := smap["port"].(string)
		channel := smap["channel"].(float64) // keep as float64, thaat's what json.Unmarshal needs
		program := smap["channel"].(float64) // keep as float64, thaat's what json.Unmarshal needs
		// NOTE: the channel value coming in here is 1-based
		m.soundOutputs[name], err = m.makeSoundOutput(uint8(channel), uint8(program))
		if err != nil {
			return fmt.Errorf("unable to make synth output for port=%s, err=%s", port, err)
		}
	}
	return nil
}

// Listen for input on a particular MIDI device
func (m *OscmidiDevice) Listen(dev string) error {
	return fmt.Errorf("oscmidi.Listen does nothing")
	/*
		devid, stream, err := m.getInputStream(dev)
		if err != nil {
			return err
		}
		m.MidiInputs[dev] = &midiInput{deviceID: devid, stream: stream}
		m.listeningNames = append(m.listeningNames, dev)
		return nil
	*/
}
