package main

import (
	"os/signal"
	"syscall"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/responder"
	_ "github.com/vizicist/palette/tool"
	"github.com/vizicist/palette/twinsys"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	r := responder.NewResponder_default()
	e := engine.NewEngine("engine")
	e.CursorManager.AddCursorResponder("default", r)
	e.Start()

	if engine.ConfigBoolWithDefault("twinsys", false) {
		twinsys.Run()
	} else {
		select {}
	}
}
