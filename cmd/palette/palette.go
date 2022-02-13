package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/vizicist/palette/block"
	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/window"
)

func usage() {
	usage := "Usage: palette {start|stop|sendlogs|api} ...\n"
	os.Stderr.WriteString(usage)
	log.Fatal(usage)
}

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	flag.Parse()
	args := flag.Args()
	engine.InitDebug()

	engine.InitLog("palette")
	engine.InitNATS()
	engine.ConnectToNATSServer()
	engine.CliCommand(args)
}

/*
func doEngine() {

	log.Printf("====================== Palette Engine is starting\n")

	engine.InitMIDI()
	engine.InitSynths()

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

	if engine.ConfigBoolWithDefault("sanitycheck", false) {
		go engine.SanityCheckForever()
	}

	if engine.ConfigBoolWithDefault("winsys", false) {
		winsys.Run() // must run in main thread, never returns
	} else {
		select {} // block forever
	}

}
*/
