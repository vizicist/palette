package main

import (
	"flag"
	"os/signal"
	"syscall"

	_ "github.com/vizicist/palette/block"
	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/window"
	"github.com/vizicist/palette/winsys"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLog("winsys")
	engine.InitDebug()

	flag.Parse()

	// go engine.StartNATSServer()

	r := engine.TheRouter()
	go r.StartNATSClient()

	// go r.StartOSC("127.0.0.1:3333")
	// go r.StartMIDI()
	// go r.StartRealtime()
	// go r.StartCursorInput()
	// go r.InputListener()

	// must run in main thread
	winsys.Run()

	select {} // block forever

}
