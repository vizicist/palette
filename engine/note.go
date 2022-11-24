package engine

import (
	"fmt"
	// _ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

type NoteOn struct {
	Synth    string
	Pitch    uint8
	Velocity uint8
}

func (n *NoteOn) String() string {
	return fmt.Sprintf("(NoteOn Synth=%s Pitch=%d)", n.Synth, n.Pitch)
}

type NoteOff struct {
	Synth    string
	Pitch    uint8
	Velocity uint8
}

func (n NoteOff) String() string {
	return fmt.Sprintf("(NoteOff Synth=%s Pitch=%d)", n.Synth, n.Pitch)
}

// NoteFull has a duration, i.e. it's a combination of a NoteOn and NoteOff
type NoteFull struct {
	Synth    string
	Duration Clicks
	Pitch    uint8
	Velocity uint8
	// msg      midi.Message
	// bytes    []byte
}

func NewNoteFull(pitch, velocity uint8, duration Clicks, synth string) *NoteFull {
	return &NoteFull{
		Synth:    synth,
		Duration: duration,
		Pitch:    pitch,
		Velocity: velocity,
		// msg:      []byte{},
		// bytes:    []byte{},
	}
}

func (n *NoteFull) String() string {
	return fmt.Sprintf("(NoteFull Synth=%s Pitch=%d Duration=%d)", n.Synth, n.Pitch, n.Duration)
}

type NoteBytes struct {
	bytes []byte
}

func NewNoteBytes(bytes []byte) *NoteBytes {
	return &NoteBytes{
		bytes: bytes,
	}
}

func (n *NoteBytes) String() string {
	return fmt.Sprintf("(NoteBytes %v)", n.bytes)
}

type NoteSystem struct {
	system byte
}

func NewNoteSystem(system byte) *NoteSystem {
	return &NoteSystem{system: system}
}

func (n *NoteSystem) String() string {
	return fmt.Sprintf("(NoteSystem byte=0%02x)", n.system)
}

type ProgramChange struct {
	Program uint8
}

func NewProgramChange(program uint8) *ProgramChange {
	return &ProgramChange{Program: program}
}

func (n *ProgramChange) String() string {
	return fmt.Sprintf("(ProgramChange program=%d)", n.Program)
}

type ChanPressure struct {
	Synth    string
	Pressure uint8
}

func NewChanPressure(pressure uint8, synth string) *ChanPressure {
	return &ChanPressure{Pressure: pressure, Synth: synth}
}

func (n *ChanPressure) String() string {
	return fmt.Sprintf("(ChanPressure pressure=%d)", n.Pressure)
}

type Controller struct {
	Controller uint8
	Value      uint8
	Synth      string
}

// NewController create a new noteoff
func NewController(controller uint8, value uint8, synth string) *Controller {
	return &Controller{Controller: controller, Value: value, Synth: synth}
}

func (n *Controller) String() string {
	return fmt.Sprintf("(Controller controller=%d value=%d)", n.Controller, n.Value)
}

type PitchBend struct {
	data1 uint8
	data2 uint8
}

func NewPitchBend(data1 uint8, data2 uint8) *PitchBend {
	return &PitchBend{data1: data1, data2: data2}
}

func (n *PitchBend) String() string {
	return fmt.Sprintf("(PitchBend data1=%d data2=%d)", n.data1, n.data2)
}
