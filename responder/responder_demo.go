package responder

import (
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
	engine.Info("NewResponder_demo in OnCursorEvent!")
	if ce.Ddu == "down" {
		go func() {
			note := r.cursorToNote(ce)
			engine.Info("cursor down: publishing", "note", note.String())
			engine.SendNoteToSynth(note)
			time.Sleep(2 * time.Second)
			note.TypeOf = "noteoff"
			engine.SendNoteToSynth(note)
		}()
	}
	engine.Info("OnCursorEvent in responder_demo called!", "ce", ce)
}

/////////////////////////// internal things

func (r *Responder_demo) cursorToNote(ce engine.CursorEvent) *engine.Note {
	pitch := int(ce.X * 126.0)
	_ = pitch
	s := "+b"
	note, err := engine.NoteFromString((s))
	if err != nil {
		engine.LogError(err)
		note, _ = engine.NoteFromString("+a")
	}
	return note
}
