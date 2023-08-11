package kit

// These are the values of MIDI status bytes
const (
	// NoteOn
	NoteOnStatus       byte = 0x90
	NoteOffStatus      byte = 0x80
	PressureStatus     byte = 0xa0
	ControllerStatus   byte = 0xb0
	ProgramStatus      byte = 0xc0
	ChanPressureStatus byte = 0xd0
	PitchbendStatus    byte = 0xe0
)
