package responder

import (
	"log"

	"github.com/vizicist/palette/engine"
)

func init() {
	engine.AddResponder("default", NewResponder_default())
}

type Responder_default struct {
	// ctx *engine.ResponderContext
}

func NewResponder_default() *Responder_default {
	p := &Responder_default{}
	return p
}

/////////////////////////// external interface

func (r *Responder_default) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	log.Printf("Responder_default.OnCursorEvent: ce=%v\n", ce)
	nt1 := r.cursorToNote(ce)
	nt2, _ := engine.NoteFromString("c")

	clicks := ctx.CurrentClick()
	ctx.ScheduleNoteNow(nt1)
	ctx.ScheduleNoteAt(nt2, clicks)

}

/////////////////////////// internal things

func (r *Responder_default) cursorToNote(ce engine.CursorEvent) *engine.Note {
	pitch := int(ce.X * 126.0)
	_ = pitch
	note, err := engine.NoteFromString("g")
	if err != nil {
		log.Printf("xyzToNote: bad note - err=%s\n", err)
		return nil
	}
	return note
}
