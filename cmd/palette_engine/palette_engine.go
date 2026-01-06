package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"runtime/pprof"
	"syscall"

	"github.com/pkg/profile"
	"github.com/vizicist/palette/kit"
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

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in palette_engine called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in palette_engine has been called")
			kit.LogError(err, "r", r, "stack", stacktrace)
		}
	}()

	kit.InitKit()
	kit.InitEngine()

	kit.StartEngine()

	go func() {
		kit.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	select {}
}
