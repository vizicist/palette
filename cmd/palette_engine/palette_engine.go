package main

import (
	"flag"
	"log"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/gui"
)

func main() {

	// signal.Ignore(syscall.SIGHUP)
	// signal.Ignore(syscall.SIGINT)

	engine.InitLogs()
	engine.InitDebug()

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	flag.Parse()

	engine.InitMIDI()

	go engine.StartNATSServer()
	go engine.StartOSC("127.0.0.1:3333")
	go engine.StartNATSClient()
	go engine.StartMIDI()
	go engine.StartRealtime()
	go engine.StartCursorInput()

	go engine.ListenForLocalDeviceInputsForever() // never returns

	gui.Run()

	// block forever
	// select {}

}
