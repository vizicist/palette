package main

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

func main() {
	engine.InitLog("app_example")

	log.Printf("app_example.go started")

	responder := engine.NewResponder()

	responder.OnCursorEvent(func(ce engine.CursorDeviceEvent) {
		if ce.Ddu == "down" {
			note := cursorToNote(ce)
			log.Printf("cursor down: publishing note=%s\n", note.String())
			err := engine.PublishNoteEvent(engine.PaletteInputEventSubject, note, "app_example")
			if err != nil {
				log.Printf("OnCursorEvent: err=%s\n", err)
			}
			note.TypeOf = "noteoff"
			err = engine.PublishNoteEvent(engine.PaletteInputEventSubject, note, "app_example")
			if err != nil {
				log.Printf("OnCursorEvent: err=%s\n", err)
			}

		}
		log.Printf("OnCursorEvent in app_example called! ce=%v\n", ce)
	})

	err := responder.RunForever()
	if err != nil {
		log.Printf("ResponderRunForever: err=%s\n", err)
	} else {
		log.Printf("ResponderRunForever: stoppeed\n")
	}
}
