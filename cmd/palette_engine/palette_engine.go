package main

import (
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/vizicist/palette/engine"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLog("engine")
	log.Printf("====================== Palette Engine is starting\n")

	engine.InitDebug()

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	flag.Parse()

	engine.InitMIDI()
	engine.InitSynths()
	engine.InitNATS()
	go engine.StartNATSServer()

	r := engine.TheRouter()

	go r.StartOSC("127.0.0.1:3333")
	go r.StartNATSClient()
	go r.StartMIDI()
	go r.StartRealtime()
	go r.StartCursorInput()

	r.InputListener() // never returns
}
