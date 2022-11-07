package engine

import (
	"fmt"
	"log"
)

//////////////////////////////////////////////////////////////////

func (note *Note) BytesString() string {
	return "BytesStringNeedsImplementing"
}

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

func NewSimpleCmd(subj string) Cmd {
	return Cmd{Subj: subj, Values: nil}
}

func NewNoteCmd(note *Note) Cmd {
	values := make(map[string]string)
	values["note"] = note.String()
	return Cmd{Subj: "note", Values: values}
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
func ToBool(arg interface{}) bool {
	r, ok := arg.(bool)
	if !ok {
		log.Printf("Unable to convert interface to bool!\n")
		r = false
	}
	return r
}

// ToString xxx
func ToString(arg interface{}) string {
	if arg == nil {
		return ""
	}
	r, ok := arg.(string)
	if !ok {
		log.Printf("Unable to convert interface to string!\n")
		r = ""
	}
	return r
}
