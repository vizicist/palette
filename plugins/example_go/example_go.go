package main

import (
	"log"

	"github.com/vizicist/palette/engine"
)

func xyzToNote(x, y, z float32) *engine.Note {
	pitch := int(x * 127.0)
	_ = x
	_ = y
	_ = z
	_ = pitch
	s := "+b"
	note, err := engine.NoteFromString((s))
	if err != nil {
		log.Printf("xyzToNote: bad note - %s\n", s)
		note, _ = engine.NoteFromString("+a")
	}
	return note
}

func callback(args map[string]string) {
	log.Printf("callback: args=%s\n", args)
	eventType := args["event"]
	if eventType == "" {
		log.Printf("callback: No event value!?\n")
		return
	}
	switch eventType {
	case "cursor_down":
		// source := args["source"]
		x, y, z, err := engine.GetArgsXYZ(args)
		if err != nil {
			log.Printf("callback: bad xyz, err=%s\n", err)
			return
		}
		note := xyzToNote(x, y, z)
		log.Printf("callback: note=%s\n", note.String())
		err = engine.PublishNoteEvent(engine.PaletteInputEventSubject, note, "example_go")
		if err != nil {
			log.Printf("callback: PlayNote err=%s\n", err)
		}
	default:
		log.Printf("callback: ignoring eventType=%s\n", eventType)
		return
	}
}

func main() {
	engine.InitLog("example_go")
	err := engine.PluginRunForever(callback)
	if err != nil {
		log.Printf("PluginRunForever: err=%s\n", err)
	} else {
		log.Printf("PluginRunForever: stoppeed\n")
	}
}
