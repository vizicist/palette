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
	engine.InitNATS()
	engine.ConnectToNATSServer()

	flag.Parse()
	out := CliCommand(flag.Args())
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
    palette list {category}
    palette load {region}.{category}.{preset}
    palette save {region}.{category}.{preset}
    palette set {region}.{category}.{parameter} {value}]
    palette get {region}.{category}.{parameter}
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

func CliCommand(args []string) string {

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

	const engineexe = "palette_engine.exe"

	switch word0 {

	case "":
		return usage()

	case "set":
		args := fmt.Sprintf("\"name\":\"%s\",\"value\":\"%s\"", word1, word2)
		return interpretApiOutput(engine.EngineAPI("value.set", args))

	case "get":
		args := fmt.Sprintf("\"name\":\"%s\"", word1)
		return interpretApiOutput(engine.EngineAPI("value.get", args))

	case "load":
		args := fmt.Sprintf("\"preset\":\"%s\"", word1)
		return interpretApiOutput(engine.EngineAPI("preset.load", args))

	case "save":
		args := fmt.Sprintf("\"preset\":\"%s\"", word2)
		return interpretApiOutput(engine.EngineAPI("preset.save", args))

	case "list":
		category := ""
		if word1 != "" {
			category = word1
		}
		args := fmt.Sprintf("\"category\":\"%s\"", category)
		return interpretApiOutput(engine.EngineAPI("preset.list", args))

	case "sendlogs":
		return CliCommand([]string{"api", "global.sendlogs"})

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
				return fmt.Sprintf("Engine not started: err=%s\n", err)
			}
			return "Engine started\n"
		} else {
			// Start a specific process
			args := fmt.Sprintf("\"process\":\"%s\"", word1)
			return interpretApiOutput(engine.EngineAPI("process.start", args))
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
			return interpretApiOutput(engine.EngineAPI("process.stop", args))
		}

	case "api":
		return interpretApiOutput(engine.EngineAPI(word1, word2))

	default:
		// NEW retmap
		return fmt.Sprintf("Unrecognized command: %s", word0)
	}
	return ""
}
