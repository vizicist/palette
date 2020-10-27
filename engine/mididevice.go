package engine

import (
	"log"
	"net"
	"time"
)

// MidiDevice mediates all access to MIDI devices
type MidiDevice interface {
	Inputs() []string
	ListeningInputs() []string
	Outputs() []string
	SendNote(note *Note, debug bool, callbacks []*NoteOutputCallback)
	SendANO(outputName string) // lame
	ReadEvent(inputName string) (MidiDeviceEvent, error)
	Poll(inputName string) (bool, error)
	Listen(inputName string) error
	LoadSounds(path string) error
}

// MidiDeviceEvent is a single MIDI event
type MidiDeviceEvent struct {
	Timestamp int64 // milliseconds
	Status    int64
	Data1     int64
	Data2     int64
}

// ListenToMIDI publishes MIDI events, and never returns unless there's an error
func ListenToMIDI() error {

	midiDevice, err := NewMidiDevice()
	if err != nil {
		log.Printf("NewMidiDevice: %s\n", err)
	}

	midiin := ConfigValue("midiinput")
	if midiin == "" {
		log.Printf("ListenToMidi: no midiinput value\n")
	} else {
		err = midiDevice.Listen(midiin)
		if err != nil {
			log.Printf("ListenToMidi: Unable to listen to midiinput=%s err=%s\n", midiin, err)
		} else {
			log.Printf("ListenToMidi: midiinput=%s\n", midiin)
		}
	}

	// source := midiin + "@" + getOutboundIP().String()

	for {
		for _, inputName := range midiDevice.ListeningInputs() {
			hasinput, err := midiDevice.Poll(inputName)
			if err != nil {
				log.Printf("midiDevice.Poll: inputName=%s err=%s\n", inputName, err)
				continue
			}
			if hasinput {
				event, err := midiDevice.ReadEvent(inputName)
				if err != nil {
					log.Printf("midiDevice.ReadEvent: err=%s\n", err)
				} else {
					err = PublishMidiDeviceEvent(event)
					if err != nil {
						log.Printf("PublishMidiDeviceEvent: err=%s\n", err)
					}
				}
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	// never returns
}

// GetOutboundIP preferred outbound ip of this machine
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
