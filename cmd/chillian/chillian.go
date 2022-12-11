package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/vizicist/palette/engine"
	agent "github.com/vizicist/palette/task"

	// _ "github.com/vizicist/palette/task"
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

	layerA := engine.NewLayer("A")
	layerB := engine.NewLayer("B")

	layerA.SetResolumeLayer(1, 3334)
	layerB.SetResolumeLayer(2, 3335)

	layerA.ApplyPreset("snap.Yellow_Spheres")
	layerB.ApplyPreset("snap.Aurora_Borealis")

	layerA.SetParam("visual.shape", "square")
	layerB.SetParam("visual.shape", "circle")

	layerA.SavePreset("visual.TestingSquare")
	layerB.SavePreset("visual.TestingCircle")

	respondDefault := agent.GetAgent("default")
	// respondLogger := agent.GetAgent("logger")
	respondChillian := agent.GetAgent("chillian")

	layerA.AttachAgent(respondChillian)
	layerB.AttachAgent(respondDefault)

	layerA.AllowSource("A")
	layerB.AllowSource("B")

	done := make(chan bool)
	e.Start(done)
	e.WaitTillDone()
	if doProfile {
		pprof.StopCPUProfile()
	}
	os.Exit(0)

	// select {}
}
