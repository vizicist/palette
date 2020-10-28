package engine

/*
// MidiDevice mediates all access to MIDI devices
type MidiDevice interface {
	Init() error
	Inputs() []string
	ListeningInputs() []string
	Outputs() []string
	SendNote(note *Note, debug bool, callbacks []*NoteOutputCallback)
	SendANO(outputName string) // lame
	ReadEvent(inputName string) (MidiDeviceEvent, error)
	Poll(inputName string) (bool, error)
	Listen(inputName string) error
}
*/

// MidiDeviceEvent is a single MIDI event
type MidiDeviceEvent struct {
	Timestamp int64 // milliseconds
	Status    int64
	Data1     int64
	Data2     int64
}
