package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/gui"
	"github.com/vizicist/palette/tools"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {

	engine.InitLogs("engine.log")
	engine.InitDebug()

	log.Printf("Palette_Engine: starting\n")

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

	doNATS := false
	if doNATS {
		// The NATS stuff is when we want to accept APIs from other machines
		// It's also used for the Python gui, since the APIS it uses need results
		go engine.StartNATSServer()
		go engine.StartNATSListener()
	}

	// OSC is used for simpler script-driven APIs that don't need results
	go engine.StartOSCListener("127.0.0.1:3333")

	go engine.StartMIDI()
	go engine.StartRealtime()
	go engine.StartGestureInput()

	go engine.ListenForLocalDeviceInputsForever()

	doGui := engine.ConfigBoolWithDefault("gui", false)
	switch doGui {
	case true:
		gui.RegisterToolType("Console", tools.NewConsole)
		gui.RegisterToolType("Riff", tools.NewRiff)
		gui.Run() // this never returns
		log.Printf("Palette_Engine: GUI has exited!?\n")
	case false:
		select {} // pause forever
	}
}
