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

	go engine.StartNATSServer()
	go engine.StartOSC("127.0.0.1:3333")
	go engine.StartNATSClient()
	go engine.StartMIDI()
	go engine.StartRealtime()
	go engine.StartCursorInput()

	engine.ListenForLocalDeviceInputsForever() // never returns
}
