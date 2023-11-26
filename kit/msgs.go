package engine

import (
	"fmt"
)

//////////////////////////////////////////////////////////////////

/*
func (note *Note) BytesString() string {
	return "BytesStringNeedsImplementing"
}
*/

/*
func NewNoteMsg(note *Note) string {
	return fmt.Sprintf(
		"\"type\":\"%s\","+
			"\"clicks\":\"%d\","+
			"\"duration\":\"%d\","+
			"\"pitch\":\"%d\","+
			"\"velocity\":\"%d\","+
			"\"sound\":\"%s\","+
			"\"bytes\":\"%s\"",
		note.TypeOf,
		note.Clicks,
		note.Duration,
		note.Pitch,
		note.Velocity,
		note.Synth,
		note.BytesString())
}
*/

func NewSimpleCmd(subj string) Cmd {
	return Cmd{Subj: subj, Values: nil}
}

func NewSetParamMsg(name string, value string) string {
	return fmt.Sprintf("\"name\":\"%s\",\"value\":\"%s\"", name, value)
}

////////////////////////////////////////////////////////////////////

// NewClickMsg xxx
func NewClickMsg(click Clicks) string {
	return fmt.Sprintf("\"click\":\"%d\"", click)
}

//////////////////////////////////////////////////////////////////

// ToBool xxx
func ToBool(arg any) bool {
	r, ok := arg.(bool)
	if !ok {
		LogWarn("Unable to convert interface to bool!")
		r = false
	}
	return r
}

// String xxx
func String(arg any) string {
	if arg == nil {
		return ""
	}
	r, ok := arg.(string)
	if !ok {
		LogWarn("Unable to convert interface to string!")
		r = ""
	}
	return r
}
