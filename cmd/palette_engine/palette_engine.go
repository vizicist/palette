package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
    "runtime/debug"

	"github.com/pkg/profile"
	"github.com/vizicist/palette/hostwin"
	"github.com/vizicist/palette/kit"
	"github.com/vizicist/palette/twinsys"
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

	host := hostwin.NewHost("engine")
	kit.RegisterHost(host)

	kit.Init()

	defer func() {
        if r := recover(); r != nil {
			stack := debug.Stack()
            fmt.Println("PANIC!? Error:\n", string(stack))
			kit.LogWarn("PANIC","error",r,"stacktrace",string(stack))
        }
    }()

	err := kit.StartEngine()
	if err != nil {
		kit.LogError(err)
		kit.LogInfo("Unable to Start Engine, cannot continue")
		os.Exit(1)
	}

	go func() {
		host.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	b, _ := kit.GetParamBool("engine.twinsys")
	if b {
		twinsys.Run()
	} else {
		select {}
	}
}
