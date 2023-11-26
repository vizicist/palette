package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/pkg/profile"
	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/tool"
	"github.com/vizicist/palette/twinsys"
)

func main() {

	doProfile := false
	if doProfile {
		// defer profile.Start(profile.ProfilePath(".")).Stop()
		// defer profile.Start(profile.BlockProfile).Stop()
		defer profile.Start(profile.MutexProfile).Stop()

		// Old original version
		// file, _ := os.Create("./cpu.pprof")
		// engine.LogIfError(pprof.StartCPUProfile(file))
	}

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLog("engine")
	engine.InitMisc()
	engine.InitEngine()

	engine.StartEngine()

	go func() {
		engine.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	b, _ := engine.GetParamBool("global.twinsys")
	if b {
		twinsys.Run()
	} else {
		select {}
	}
}
