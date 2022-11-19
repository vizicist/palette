package engine

/*
import (
	"fmt"

	"gitlab.com/gomidi/midi/v2/drivers"
)

type Input struct {
	intype    string // "midi", "cursor"
	name      string // ""
	midiInput drivers.In
	morph     *oneMorph
}

var AllInputs = make(map[string]*Input)

func init() {
	AllInputs["morph A"] = &Input{
		intype:    "cursor",
		midiInput: nil,
		morph:     nil,
	}
	AllInputs["morph B"] = &Input{
		intype:    "cursor",
		midiInput: nil,
		morph:     nil,
	}
	AllInputs["midi"] = &Input{
		intype:    "midi",
		midiInput: nil,
		morph:     nil,
	}
}

func NewInput() *Input {
	return &Input{
		intype:    "",
		name:      "",
		midiInput: nil,
		morph:     &oneMorph{},
	}
}

func GetInput(name string) (*Input, error) {
	in, ok := AllInputs[name]
	if !ok {
		return nil, fmt.Errorf("GetInput: no such input - %s", name)
	} else {
		return in, nil
	}
}

func NewMorphInput() *Input {
	return &Input{
		intype: "morph",
		name:   "morph A",
	}
}

*/
