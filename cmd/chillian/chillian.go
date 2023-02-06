package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vizicist/palette/engine"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	e := engine.TheEngine

	e.StartPlugin("chillian")

	e.Start()
	e.WaitTillDone()
	os.Exit(0)
}
