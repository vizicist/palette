package main

import (
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/vizicist/palette/engine"
)

var MyPrefix byte = 0x55
var EraeWidth int = 0x2a
var EraeHeight int = 0x18

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitMIDI()

	flag.Parse()

	r := engine.TheRouter()
	go r.StartMIDI()
	go r.InputListener() // never returns

	args := flag.Args()
	nargs := len(args)

	cmd := "test" // default cmd
	if nargs != 0 {
		cmd = args[0]
	}
	switch cmd {
	case "test":
		engine.EraeTest()
	default:
		log.Printf("Unrecognized: %s\n", args[0])
	}
	log.Printf("Blocking forever....\n")
	select {} // block forever
}
