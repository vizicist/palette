package main

import (
	"flag"
	"log"
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

	engine.InitLog("engine")
	engine.InitDebug()
	engine.InitProcessInfo()

	log.Printf("====================== Palette Engine is starting\n")

	flag.Parse()

	// Normally, the engine should never die, but if it does,
	// other processes (e.g. resolume, vsthost) may be left around.
	// So, unless told otherwise, we kill everything to get a clean start.
	if engine.ConfigBoolWithDefault("killonstartup", true) {
		engine.KillProcess("resolume")
		engine.KillProcess("vsthost")
		engine.KillProcess("gui")
		mmtt := engine.ConfigStringWithDefault("mmtt", "")
		if mmtt != "" {
			engine.KillProcess("mmtt_" + mmtt)
		}
	}

	engine.InitMIDI()
	engine.InitSynths()
	// engine.InitNATS()
	go engine.StartNATSServer()

	r := engine.TheRouter()

	go r.StartOSC("127.0.0.1:3333")
	go r.StartNATSClient()
	go r.StartMIDI()
	go r.StartRealtime()
	go r.StartCursorInput()
	go r.InputListener()

	if engine.ConfigBoolWithDefault("depth", false) {
		go engine.DepthRunForever()
	}

	if engine.ConfigBoolWithDefault("winsys", false) {
		// must run in main thread
		winsys.Run()
		// it only returns when the window is closed.
	}
	select {} // block forever

}
