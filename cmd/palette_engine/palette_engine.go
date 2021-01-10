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
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {

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

	doGui := engine.ConfigBoolWithDefault("gui", false)
	switch doGui {
	case true:
		gui.Run() // this never returns
		log.Printf("GUI has exited!?\n")
	case false:
		select {} // pause forever
	}
}
