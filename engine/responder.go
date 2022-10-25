package engine

import (
	"log"
)

type Responder interface {
	OnCursorDeviceEvent(e CursorDeviceEvent)
}

var Responders = make(map[string]Responder)

func AddResponder(name string, r Responder) {
	log.Printf("AddResponder: r=%v\n", r)
	Responders[name] = r
}

func CallResponders(e CursorDeviceEvent) {
	for name, responder := range Responders {
		log.Printf("CallResponders: name=%s\n", name)
		responder.OnCursorDeviceEvent(e)
	}
}
