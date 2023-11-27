package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/pkg/profile"
	"github.com/vizicist/palette/kit"
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
		// kit.LogIfError(pprof.StartCPUProfile(file))
	}

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	kit.InitLog("engine")
	kit.InitMisc()
	kit.InitEngine()

	kit.StartEngine()

	go func() {
		kit.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	b, _ := kit.GetParamBool("global.twinsys")
	if b {
		twinsys.Run()
	} else {
		select {}
	}
}
