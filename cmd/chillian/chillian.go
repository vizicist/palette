package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/plugin"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	e := engine.TheEngine()

	e.StartPlugin("chillian")

	e.Start()
	e.WaitTillDone()
	os.Exit(0)
}
