package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
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

	regionPtr := flag.String("region", "*", "Region name or *")

	flag.Parse()
	out := CliCommand(*regionPtr, flag.Args())
	// If the output starts with [ or {,
	// we assume it's a json array or object,
	// and print it out more readably.
	if out != "" {
		switch out[0] {
		case '[':
			out = readableArray(out)
		case '{':
			out = readableObject(out)
		}
	}
	os.Stdout.WriteString(out)
}

func usage() string {
	return `Usage:
    palette start [process]
    palette stop [process]
    palette sendlogs
    palette list [{category}]
    palette [-region={region}] load {category}.{preset}
    palette [-region {region}] save {category}.{preset}
    palette [-region {region}] set {category}.{parameter} [{value}]
    palette [-region {region}] get {category}.{parameter}
    palette api {api} {args}`
}

func readableArray(s string) string {
	var arr []string
	err := json.Unmarshal([]byte(s), &arr)
	if err != nil {
		return fmt.Sprintf("Unable to print API array output? err=%s s=%s", err, s)
	}
	sort.Strings(arr)
	out := ""
	for _, val := range arr {
		out += fmt.Sprintf("%s\n", val)
	}
	return out
}

func readableObject(s string) string {
	objmap, err := engine.StringMap(s)
	if err != nil {
		return fmt.Sprintf("Unable to parse json: %s", s)
	}

	arr := make([]string, 0, len(objmap))
	for key, val := range objmap {
		arr = append(arr, fmt.Sprintf("%s %s\n", key, val))
	}
	sort.Strings(arr)
	out := ""
	for _, val := range arr {
		out += fmt.Sprintf("%s\n", val)
	}
	return out
}

// interpretApiOutput takes the result of an API invocation and
// produces what will appear in visible output from a CLI command.
func interpretApiOutput(rawresult string, err error) string {
	if err != nil {
		return fmt.Sprintf("Internal error: %s", err)
	}
	retmap, err2 := engine.StringMap(rawresult)
	if err2 != nil {
		return fmt.Sprintf("API produced non-json output: %s", rawresult)
	}
	e, eok := retmap["error"]
	if eok {
		return fmt.Sprintf("API error: %s", e)
	}
	result, rok := retmap["result"]
	if !rok {
		return "API produced no error or result"
	}
	return result
}

func CliCommand(region string, args []string) string {

	if len(args) == 0 {
		return usage()
	}

	const engineexe = "palette_engine.exe"

	switch args[0] {

	case "load", "save":
		if len(args) < 1 {
			return "Insufficient arguments"
		}
		apiargs := fmt.Sprintf("\"region\":\"%s\",\"preset\":\"%s\"", region, args[1])
		return interpretApiOutput(engine.EngineAPI(args[0], apiargs))

	case "get":
		if len(args) < 1 {
			return "Insufficient arguments"
		}
		apiargs := fmt.Sprintf("\"region\":\"%s\",\"name\":\"%s\"", region, args[1])
		return interpretApiOutput(engine.EngineAPI(args[0], apiargs))

	case "set":
		if len(args) < 2 {
			return "Insufficient arguments"
		}
		apiargs := fmt.Sprintf("\"region\":\"%s\",\"name\":\"%s\",\"value\":\"%s\"", region, args[1], args[2])
		return interpretApiOutput(engine.EngineAPI(args[0], apiargs))

	case "list":
		category := "*"
		if len(args) > 1 {
			category = args[1]
		}
		args := fmt.Sprintf("\"category\":\"%s\"", category)
		return interpretApiOutput(engine.EngineAPI("list", args))

	case "sendlogs":
		return interpretApiOutput(engine.EngineAPI("global.sendlogs", ""))

	case "status", "tasks":
		return engine.ProcessStatus()

	case "version":
		return engine.PaletteVersion()

	case "activate":
		return interpretApiOutput(engine.EngineAPI("activate", ""))

	case "start":
		process := "engine"
		if len(args) > 1 {
			process = args[1]
		}
		if process == "engine" {

			// Kill any currently-running engine.
			// The other processes will be killed by
			// the engine when it starts up.
			engine.KillExecutable(engineexe)

			// Start the engine (which also starts up other processes)
			fullexe := filepath.Join(engine.PaletteDir(), "bin", engineexe)
			err := engine.StartExecutableLogOutput("engine", fullexe, true, "")
			if err != nil {
				return fmt.Sprintf("Engine not started: err=%s\n", err)
			}
			return "Engine started\n"
		} else {
			// Start a specific process
			args := fmt.Sprintf("\"process\":\"%s\"", process)
			return interpretApiOutput(engine.EngineAPI("start", args))
		}

	case "stop":
		process := "engine"
		if len(args) > 1 {
			process = args[1]
		}
		if process == "all" {
			engine.StopRunning("all")
			engine.KillExecutable(engineexe)
		} else if process == "engine" {
			// first stop everything else, unless killonstartup is false
			if engine.ConfigBoolWithDefault("killonstartup", true) {
				engine.StopRunning("all")
			}
			// then kill ourselves
			engine.KillExecutable(engineexe)
		} else {
			args := fmt.Sprintf("\"process\":\"%s\"", process)
			return interpretApiOutput(engine.EngineAPI("stop", args))
		}

	case "api":
		if len(args) < 2 {
			return "Insufficient arguments"
		}
		return interpretApiOutput(engine.EngineAPI(args[1], args[2]))

	default:
		// NEW retmap
		return fmt.Sprintf("Unrecognized command: %s", args[0])
	}
	return ""
}
