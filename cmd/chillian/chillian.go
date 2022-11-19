package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/tool"
)

func main() {

	doProfile := false
	if doProfile {
		file, _ := os.Create("./cpu.pprof")
		pprof.StartCPUProfile(file)
	}

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	e := engine.TheEngine()

	playerA := engine.NewPlayer("A")
	playerB := engine.NewPlayer("B")

	playerA.SetResolumeLayer(1, 3334)
	playerB.SetResolumeLayer(2, 3335)

	playerA.ApplyPreset("snap.Yellow_Spheres")
	playerB.ApplyPreset("snap.Aurora_Borealis")

	playerA.SetParam("visual.shape", "square")
	playerB.SetParam("visual.shape", "circle")

	playerA.SavePreset("visual.TestingSquare")
	playerB.SavePreset("visual.TestingCircle")

	respondDefault := engine.GetResponder("default")
	respondLogger := engine.GetResponder("logger")
	respondChillian := engine.GetResponder("chillian")

	playerA.AttachResponder(respondDefault)
	playerA.AttachResponder(respondLogger)

	playerB.AttachResponder(respondChillian)

	playerA.AllowSource("A")
	playerB.AllowSource("B")

	go func() {
		e.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	e.Start()
	select {}
}
