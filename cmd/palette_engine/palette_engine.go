package main

import (
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"

	_ "github.com/vizicist/palette/agent"
	"github.com/vizicist/palette/engine"
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

	// Eventually the defult should be ""
	agents := engine.ConfigStringWithDefault("agents", "ppro")
	for _, agentName := range strings.Split(agents, ",") {
		e.StartAgent(agentName)
	}

	done := make(chan bool)
	e.Start(done)

	go func() {
		e.WaitTillDone()
		if doProfile {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()

	if engine.ConfigBoolWithDefault("twinsys", false) {
		twinsys.Run()
	} else {
		select {}
	}
}
