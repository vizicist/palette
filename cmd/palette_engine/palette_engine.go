package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/responder"
	_ "github.com/vizicist/palette/tool"
	"github.com/vizicist/palette/twinsys"
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

	players := map[string]*engine.Player{}
	presets := map[string]string{
		"A": "snap.Yellow_Spheres",
		"B": "snap.Blue_Spheres",
		"C": "snap.Orange_Spheres",
		"D": "snap.Pink_Spheres",
	}
	shapes := map[string]string{
		"A": "square",
		"B": "circle",
		"C": "triangle",
		"D": "line",
	}

	respondDefault := engine.GetResponder("default")

	for n, playerName := range []string{"A", "B", "C", "D"} {
		p := engine.NewPlayer(playerName)
		p.SetResolumeLayer(1+n, 3334+n)
		p.ApplyPreset(presets[playerName])
		p.SetParam("visual.shape", shapes[playerName])
		p.SavePreset(presets[playerName] + "_test")
		p.AttachResponder(respondDefault)
		p.AllowSource(playerName)
		players[playerName] = p
	}

	go func() {
		e.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	e.Start()

	if engine.ConfigBoolWithDefault("twinsys", false) {
		twinsys.Run()
	} else {
		select {}
	}
}
