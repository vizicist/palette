package responder

import (
	"log"

	"github.com/vizicist/palette/engine"
)

func init() {
	engine.AddResponder("default", NewResponder_default())
}

type Responder_default struct {
}

func NewResponder_default() *Responder_default {
	p := &Responder_default{}
	return p
}

/////////////////////////// external interface

func (r *Responder_default) OnCursorEvent(ce engine.CursorEvent, cm *engine.ResponderManager) {
	log.Printf("Responder_default.OnCursorEvent: ce=%v\n", ce)
	// nt := r.cursorToNote(ce)

}

/////////////////////////// internal things

func (r *Responder_default) cursorToNote(ce engine.CursorEvent) *engine.Note {
	pitch := int(ce.X * 126.0)
	_ = pitch
	s := "+b"
	note, err := engine.NoteFromString((s))
	if err != nil {
		log.Printf("xyzToNote: bad note - %s\n", s)
		note, _ = engine.NoteFromString("+a")
	}
	return note
}
