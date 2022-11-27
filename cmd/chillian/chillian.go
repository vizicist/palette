package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/responder"

	// _ "github.com/vizicist/palette/responder"
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

	respondDefault := responder.GetResponder("default")
	// respondLogger := responder.GetResponder("logger")
	respondChillian := responder.GetResponder("chillian")

	playerA.AttachResponder(respondChillian)
	playerB.AttachResponder(respondDefault)

	playerA.AllowSource("A")
	playerB.AllowSource("B")

	done := make(chan bool)
	e.Start(done)
	e.WaitTillDone()
	if doProfile {
		pprof.StopCPUProfile()
	}
	os.Exit(0)

	// select {}
}
