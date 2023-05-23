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

	engine.InitLog("engine")
	engine.InitMisc()
	engine.InitEngine()

	engine.Start()

	go func() {
		engine.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	if engine.GetParamBool("engine.twinsys") {
		twinsys.Run()
	} else {
		select {}
	}
}
