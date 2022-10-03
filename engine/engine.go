package engine

import (
	"flag"
	"log"

	"github.com/vizicist/palette/twinsys"
)

type Engine struct {
	Router *Router
	stopme bool
}

var TheEngine *Engine

func RunEngine() {

	TheEngine = &Engine{
		Router: TheRouter(),
		stopme: false,
	}

	InitLog("engine")
	InitDebug()
	InitProcessInfo()

	log.Printf("====================== Palette Engine is starting\n")

	flag.Parse()

	// Normally, the engine should never die, but if it does,
	// other processes (e.g. resolume, bidule) may be left around.
	// So, unless told otherwise, we kill everything to get a clean start.
	if ConfigBoolWithDefault("killonstartup", true) {
		KillAll()
	}

	InitMIDI()
	InitSynths()
	// InitNATS()
	go StartNATSServer()

	r := TheEngine.Router
	go r.StartOSC("127.0.0.1:3333")
	go r.StartNATSClient()
	go r.StartMIDI()
	go r.StartRealtime()
	go r.StartCursorInput()
	go r.InputListener()

	if ConfigBoolWithDefault("depth", false) {
		go DepthRunForever()
	}

	if ConfigBoolWithDefault("twinsys", false) {
		// must run in main thread
		twinsys.Run()
		// it only returns when the window is closed.
	}
	select {} // block forever
}
