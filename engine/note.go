package engine

import (
	"fmt"
)

type NoteOn struct {
	// Channel  uint8
	Synth    *Synth
	Pitch    uint8
	Velocity uint8
}

func (n *NoteOn) String() string {
	return fmt.Sprintf("(NoteOn Synth=%s Pitch=%d)", n.Synth.name, n.Pitch)
}

type NoteOff struct {
	// Channel  uint8
	Synth    *Synth
	Pitch    uint8
	Velocity uint8
}

func (n NoteOff) String() string {
	return fmt.Sprintf("(NoteOff Synth=%s Pitch=%d)", n.Synth.name, n.Pitch)
}

// NoteFull has a duration, i.e. it's a combination of a NoteOn and NoteOff
type NoteFull struct {
	// Channel  uint8
	Synth     *Synth
	Duration  Clicks
	SynthName string
	Pitch     uint8
	Velocity  uint8
	// msg      midi.Message
	// bytes    []byte
}

func NewNoteFull(synth *Synth, pitch, velocity uint8, duration Clicks) *NoteFull {
	return &NoteFull{
		// Channel:  channel,
		Synth:    synth,
		Duration: duration,
		Pitch:    pitch,
		Velocity: velocity,
	}
}

func (n *NoteFull) String() string {
	return fmt.Sprintf("(NoteFull Synth=%s Pitch=%d Duration=%d)", n.Synth.name, n.Pitch, n.Duration)
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
	Channel uint8
	Program uint8
}

func NewProgramChange(channel, program uint8) *ProgramChange {
	return &ProgramChange{Channel: channel, Program: program}
}

func (n *ProgramChange) String() string {
	return fmt.Sprintf("(ProgramChange program=%d)", n.Program)
}

type ChanPressure struct {
	Channel  uint8
	Pressure uint8
}

func NewChanPressure(channel, pressure uint8) *ChanPressure {
	return &ChanPressure{Pressure: pressure, Channel: channel}
}

func (n *ChanPressure) String() string {
	return fmt.Sprintf("(ChanPressure pressure=%d)", n.Pressure)
}

type Controller struct {
	Controller uint8
	Value      uint8
	Channel    uint8
}

// NewController create a new noteoff
func NewController(channel, controller, value uint8) *Controller {
	return &Controller{Controller: controller, Value: value, Channel: channel}
}

func (n *Controller) String() string {
	return fmt.Sprintf("(Controller controller=%d value=%d)", n.Controller, n.Value)
}

type PitchBend struct {
	data0 uint8
	data1 uint8
	data2 uint8
}

func NewPitchBend(data0, data1, data2 uint8) *PitchBend {
	return &PitchBend{data0: data0, data1: data1, data2: data2}
}

func (n *PitchBend) String() string {
	return fmt.Sprintf("(PitchBend data1=%d data2=%d)", n.data1, n.data2)
}
