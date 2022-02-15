package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/vizicist/palette/block"
	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/window"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	flag.Parse()
	args := flag.Args()
	engine.InitDebug()

	engine.InitLog("palette")
	engine.InitNATS()
	engine.ConnectToNATSServer()
	retmap := engine.CliCommand(args)
	result, rok := retmap["result"]
	errout, eok := retmap["error"]
	if eok && errout != "" {
		os.Stdout.WriteString("error: " + errout + "\n")
	} else if rok && result != "" {
		os.Stdout.WriteString("result: " + result)
	}
}
