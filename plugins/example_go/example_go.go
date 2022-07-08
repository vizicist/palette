package main

import (
	"log"

	"github.com/vizicist/palette/engine"
)

func callback(eventType string, eventData interface{}) {
	log.Printf("example_go: callback type=%s\n", eventType)
	switch data := eventData.(type) {
	case *engine.Note:
		log.Printf("example_go: GOT NOTE! pitch=%d\n", data.Pitch)
		log.Printf("example_go: data=%+v\n", *data)
		callbackNote(data)
	default:
		log.Printf("example_go: UNRECOGNIZED DATA ! data=%+v\n", eventData)
	}
}

func callbackNote(note *engine.Note) {
	log.Printf("callbackNote: source=%s sound=%s\n", note.Source, note.Sound)
}

func main() {
	engine.InitLog("example_go")
	log.Printf("plugin: started mynuid=%s\n", engine.MyNUID())
	err := engine.PluginRegister(engine.MyNUID(), "hello", "all", callback)
	log.Printf("plugin: after PluginRegister\n")
	if err != nil {
		log.Printf("plugin.Register: err=%s\n", err)
		return
	}
	log.Printf("plugin: blocking forever\n")
	select {} // block forever
}
