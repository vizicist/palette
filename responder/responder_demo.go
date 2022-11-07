package responder

import (
	"log"
	"time"

	"github.com/vizicist/palette/engine"
)

type Responder_demo struct {
}

func NewResponder_demo() *Responder_demo {
	p := &Responder_demo{}
	return p
}

/////////////////////////// external interface

func (r *Responder_demo) OnCursorEvent(ce engine.CursorEvent) {
	log.Printf("NewResponder_demo in OnCursorEvent!\n")
	if ce.Ddu == "down" {
		go func() {
			note := r.cursorToNote(ce)
			log.Printf("cursor down: publishing note=%s\n", note.String())
			engine.SendNoteToSynth(note)
			time.Sleep(2 * time.Second)
			note.TypeOf = "noteoff"
			engine.SendNoteToSynth(note)
		}()
	}
	log.Printf("OnCursorEvent in responder_demo called! ce=%v\n", ce)
}

/////////////////////////// internal things

func (r *Responder_demo) cursorToNote(ce engine.CursorEvent) *engine.Note {
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
