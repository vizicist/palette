package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/vizicist/palette/egui"
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

	profile := false
	if profile {
		f, err := os.Create("cpu.prof")
		if err != nil {
			log.Fatal(err)
		}

		err = pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	engine.InitMIDI()

	go engine.StartNATSServer()
	go engine.StartOSC("127.0.0.1:3333")
	go engine.StartNATSClient()
	go engine.StartMIDI()
	go engine.StartRealtime()
	go engine.StartCursorInput()

	go engine.ListenForLocalDeviceInputsForever()

	guitype := "ebiten"
	switch guitype {
	case "ebiten":
		egui.Run()
	case "nano":
		// GUI must run in the main thread, not in a goroutine
		gui.Run() // this never returns
	default:
		select {} // pause forever
	}
	log.Printf("GUI has exited!?\n")
}
