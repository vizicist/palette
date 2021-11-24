package main

import (
	"flag"
	"log"
	"os/signal"
	"syscall"

	_ "github.com/vizicist/palette/block"
	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/window"
	"github.com/vizicist/palette/winsys"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLog("engine")
	log.Printf("====================== Palette Engine is starting\n")

	engine.InitDebug()

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
	go r.InputListener()

	if engine.ConfigBoolWithDefault("winsys", false) {
		winsys.Run() // must run in main thread, never returns
	} else {
		select {} // block forever
	}

}
