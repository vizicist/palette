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

	engine.InitLog("erae")
	engine.InitMIDI()

	flag.Parse()

	e := engine.TheEngine()
	go e.StartMIDI()
	go e.InputListener()

	log.Printf("Blocking forever....\n")
	select {} // block forever
}
