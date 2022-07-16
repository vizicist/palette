package main

import (
	"log"

	"github.com/vizicist/palette/engine"
)

var Plugins = make(map[string]*engine.Plugin)

func callback(pluginid string, eventType string, eventData interface{}) {
	plugin, ok := Plugins[pluginid]
	if !ok {
		log.Printf("callback: no plugin %s\n", pluginid)
		return
	}
	switch data := eventData.(type) {
	case *engine.Note:
		note := data
		log.Printf("callbackNote: source=%s sound=%s\n", note.Source, note.Sound)
		note.Pitch++
		err := plugin.PlayNote(note)
		if err != nil {
			log.Printf("example_go: PlayNote err=%s\n", err)
		}
	default:
		log.Printf("example_go: UNRECOGNIZED DATA ! data=%+v\n", eventData)
	}
}

func main() {
	engine.InitLog("example_go")
	plugin, err := engine.PluginRegister("all", callback)
	if err != nil {
		log.Printf("Unable to register plugin?\n")
		return
	}
	Plugins[plugin.ID()] = plugin
	log.Printf("plugin: registered pluginid=%s\n", plugin.ID())
	if err != nil {
		log.Printf("plugin.Register: err=%s\n", err)
		return
	}
	log.Printf("plugin: blocking forever\n")
	select {} // block forever
}
