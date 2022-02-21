package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	_ "github.com/vizicist/palette/block"
	"github.com/vizicist/palette/engine"
	_ "github.com/vizicist/palette/window"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLog("palette")
	engine.InitDebug()
	engine.InitProcessInfo()
	engine.InitNATS()
	engine.ConnectToNATSServer()

	flag.Parse()
	retmap := CliCommand(flag.Args())

	result, rok := retmap["result"]
	errout, eok := retmap["error"]
	if eok && errout != "" {
		os.Stdout.WriteString("error: " + errout + "\n")
	} else if rok && result != "" {
		os.Stdout.WriteString("result: " + result)
	}
}

func CliCommand(args []string) map[string]string {

	retmap := map[string]string{}

	nargs := len(args)
	var word0, word1, word2 string
	if nargs > 0 {
		word0 = args[0]
	}
	if nargs > 1 {
		word1 = args[1]
	}
	if nargs > 2 {
		word2 = args[2]
	}

	var msg string
	const engineexe = "palette_engine.exe"

	switch word0 {

	case "":
		msg = "usage:  palette start [process]\n"
		msg += "       palette stop [process]\n"
		msg += "       palette sendlogs\n"
		msg += "       palette api {api} {args}\n"

	case "sendlogs":
		retmap = CliCommand([]string{"api", "global.sendlogs"})

	case "start":
		if word1 == "" || word1 == "engine" {

			// Kill any currently-running engine.
			// The other processes will be killed by
			// the engine when it starts up.
			engine.KillExecutable(engineexe)

			// Start the engine (which also starts up other processes)
			fullexe := filepath.Join(engine.PaletteDir(), "bin", engineexe)
			_, err := engine.StartExecutableLogOutput("engine", fullexe, true, "")
			if err != nil {
				retmap["error"] = fmt.Sprintf("start: err=%s\n", err)
			}
		} else {
			// Start a specific process
			// engine.StartRunning(word1)
			args := fmt.Sprintf("\"process\":\"%s\"", word1)
			_, err := engine.EngineAPI("process.start", args)
			if err != nil {
				retmap["error"] = fmt.Sprintf("start: process=%s err=%s\n", word1, err)
			}
		}

	case "stop":
		if word1 == "" || word1 == "engine" {
			// first stop everything else
			engine.StopRunning("all")
			// then kill ourselves
			engine.KillExecutable(engineexe)
		} else {
			// engine.StopRunning(word1)
			args := fmt.Sprintf("\"process\":\"%s\"", word1)
			_, err := engine.EngineAPI("process.stop", args)
			if err != nil {
				retmap["error"] = fmt.Sprintf("start: process=%s err=%s\n", word1, err)
			}
		}

	case "api":
		var e error
		retmap, e = engine.EngineAPI(word1, word2)
		if e != nil {
			// internal error
			log.Printf("api: error=%s\n", e)
			// NEW retmap
			retmap = map[string]string{}
			retmap["error"] = fmt.Sprintf("internal err=%s\n", e)
		}

	default:
		// NEW retmap
		retmap = map[string]string{}
		retmap["error"] = fmt.Sprintf("cliCommand: unrecognized %s\n", word0)
	}

	return retmap
}
