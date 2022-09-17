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
	engine.InitLog("example_go")

	log.Printf("example_go.go started")

	plugin := engine.NewPlugin()

	plugin.OnCursorEvent(func(ce engine.CursorDeviceEvent) {
		if ce.Ddu == "down" {
			note := cursorToNote(ce)
			log.Printf("cursor down: publishing note=%s\n", note.String())
			err := engine.PublishNoteEvent(engine.PaletteInputEventSubject, note, "example_go")
			if err != nil {
				log.Printf("OnCursorEvent: err=%s\n", err)
			}
			note.TypeOf = "noteoff"
			err = engine.PublishNoteEvent(engine.PaletteInputEventSubject, note, "example_go")
			if err != nil {
				log.Printf("OnCursorEvent: err=%s\n", err)
			}

		}
		log.Printf("OnCursorEvent in example_go called! ce=%v\n", ce)
	})

	err := plugin.RunForever()
	if err != nil {
		log.Printf("PluginRunForever: err=%s\n", err)
	} else {
		log.Printf("PluginRunForever: stoppeed\n")
	}
}
