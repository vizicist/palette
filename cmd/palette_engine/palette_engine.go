package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/tool"
	"github.com/vizicist/palette/twinsys"
)

func main() {

	doProfile := false
	if doProfile {
		file, _ := os.Create("./cpu.pprof")
		engine.LogIfError(pprof.StartCPUProfile(file))
	}

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.Init("engine")

	e := engine.NewEngine()

	e.Start()

	go func() {
		e.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	if engine.ParamBool("engine.twinsys") {
		twinsys.Run()
	} else {
		select {}
	}
}
