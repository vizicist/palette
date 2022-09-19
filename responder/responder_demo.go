package responder

import (
	"log"

	"github.com/vizicist/palette/engine"
)

func cursorToNote(ce engine.CursorDeviceEvent) *engine.Note {
	pitch := int(ce.X * 127.0)
	_ = pitch
	s := "+b"
	note, err := engine.NoteFromString((s))
	if err != nil {
		log.Printf("xyzToNote: bad note - %s\n", s)
		note, _ = engine.NoteFromString("+a")
	}
	return note
}

func NewResponder_demo() *Responder {

	log.Printf("NewResponder_demo called\n")

	responder := NewResponder()

	responder.OnCursorEvent(func(ce engine.CursorDeviceEvent) {
		log.Printf("NewResponder_demo in OnCursorEvent!\n")
		if ce.Ddu == "down" {
			note := cursorToNote(ce)
			log.Printf("cursor down: publishing note=%s\n", note.String())
			err := engine.PublishNoteEvent(engine.PaletteInputEventSubject, note, "responder_demo")
			if err != nil {
				log.Printf("OnCursorEvent: err=%s\n", err)
			}
			note.TypeOf = "noteoff"
			err = engine.PublishNoteEvent(engine.PaletteInputEventSubject, note, "responder_demo")
			if err != nil {
				log.Printf("OnCursorEvent: err=%s\n", err)
			}

		}
		log.Printf("OnCursorEvent in responder_demo called! ce=%v\n", ce)
	})

	return responder
}

/*
	err := responder.RunForever()
	if err != nil {
		log.Printf("PluginRunForever: err=%s\n", err)
	} else {
		log.Printf("PluginRunForever: stoppeed\n")
	}
*/
