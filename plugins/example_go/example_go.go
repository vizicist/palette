package main

import (
	"log"

	"github.com/vizicist/palette/engine"
)

func callback(eventType string, eventData string) {
	log.Printf("hello: callback type=%s data=%s\n", eventType, eventData)
}

func main() {
	err := engine.RegisterPlugin("hello", "all", callback)
	if err != nil {
		log.Printf("plugin.Register: err=%s\n", err)
		return
	}
	select {} // block forever
}
