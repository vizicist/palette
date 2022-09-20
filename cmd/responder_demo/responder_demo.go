package main

import (
	"log"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/responder"
)

func main() {
	engine.InitLog("responder_demo")

	log.Printf("responder_demo.go started")

	responder := responder.NewResponder_demo()
	err := responder.RunForever()
	if err != nil {
		log.Printf("ResponderRunForever: err=%s\n", err)
	} else {
		log.Printf("ResponderRunForever: stoppeed\n")
	}
}
