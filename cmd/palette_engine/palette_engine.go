package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/pkg/profile"
	"github.com/vizicist/palette/twinsys"
	"github.com/vizicist/palette/kit"
	"github.com/vizicist/palette/hostwin"
	_ "github.com/vizicist/palette/twintool"
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

	kit.TheHost = hostwin.NewHost("engine")

	// host.Init()
	// host.InitMisc()
	// host.InitEngine()
	kit.TheHost.Start()

	go func() {
		host.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	b, _ := host.GetParamBool("engine.twinsys")
	if b {
		twinsys.Run()
	} else {
		select {}
	}
}
