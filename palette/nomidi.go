// +build !windows

package palette

import (
	"fmt"
	"log"
)

// NewMidiDevice xxx
func NewMidiDevice() (MidiDevice, error) {
	return NewNomidiDevice()
}

// NomidiDevice is used when you don't want to do MIDI
type NomidiDevice struct {
}

type midiInput struct {
}

type synthOutput struct {
}

// NomidiEvent is a single MIDI input event
// XXX - since is local, PortmidiEvent should just be portmidi.Event
type NomidiEvent struct {
}

// NewNomidiDevice returns a MidiDevice interface for portmidi
func NewNomidiDevice() (MidiDevice, error) {

	pm := &NomidiDevice{}

	log.Printf("Nomidi has been initialized\n")
	return pm, nil
}

func (m NomidiDevice) LoadSounds(path string) error {
	log.Printf("NomidiDevice.LoadSounds does nothing\n")
	return nil
}

// ReadEvent xxx
func (m NomidiDevice) ReadEvent(inputName string) (MidiDeviceEvent, error) {
	return MidiDeviceEvent{}, fmt.Errorf("NomidiDevice can't do ReadEvent")
}

// Poll xxx
func (m NomidiDevice) Poll(inputName string) (bool, error) {
	return false, fmt.Errorf("NomidiDevice.Poll can't do Poll")
}

func (m *midiInput) Poll() (bool, error) {
	return false, fmt.Errorf("NomidiDevice.midiInput.Poll can't do Poll")
}

func (m *midiInput) ReadEvent() (NomidiEvent, error) {
	return NomidiEvent{}, fmt.Errorf("NomidiDevice can't do ReadEvent")
}

// SendANO sends all-notes-off
func (m *NomidiDevice) SendANO(synth string) {
	log.Printf("NomidiDevice.SendANO\n")
}

// Inputs returns a list of MIDI input names
func (m *NomidiDevice) Inputs() []string {
	return []string{}
}

// ListeningInputs returns a list of MIDI input names that are open for listening
func (m *NomidiDevice) ListeningInputs() []string {
	return []string{}
}

// Outputs returns a list of MIDI output names
func (m *NomidiDevice) Outputs() []string {
	return []string{}
}

// SendNote sends MIDI output for a Note
func (m *NomidiDevice) SendNote(n *Note, debug bool, callbacks []*NoteOutputCallback) {
	log.Printf("NomidiDevice SendNote\n")
}

// Listen for input on a particular MIDI device
func (m *NomidiDevice) Listen(dev string) error {
	return fmt.Errorf("No midi device to listen to")
}
