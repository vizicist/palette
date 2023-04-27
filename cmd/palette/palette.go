package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/vizicist/palette/engine"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	flag.Parse()

	engine.InitLog("palette")
	engine.LogInfo("Palette InitLog", "args", flag.Args())

	out := CliCommand(flag.Args())
	os.Stdout.WriteString(out)
}

func usage() string {
	return `Commands:
	palette start [ all | engine | gui | bidule | resolume ]
	palette stop [ all | engine | gui | bidule | resolume ]
	palette version
	palette {category}.{api} [ {argname} {argvalue} ] ...
	`
}

// humanReadableApiOutput takes the result of an API invocation and
// produces what will appear in visible output from a CLI command.
func humanReadableApiOutput(output map[string]string) string {
	e, eok := output["error"]
	if eok {
		return fmt.Sprintf("Error: api err=%s", e)
	}
	result, rok := output["result"]
	if !rok {
		return "Error: unexpected - no result or error in API output?"
	}
	if result == "" {
		result = "OK"
	}

	return result
}

func CliCommand(args []string) string {

	var err error

	if len(args) == 0 {
		return usage()
	}

	api := args[0]
	var arg1 string
	if len(args) > 1 {
		arg1 = args[1]
	}

	switch api {

	case "status":
		if engine.IsEngineRunning() {
			return "Engine is running."
		} else {
			return "Engine is stopped."
		}

	case "start":

		switch arg1 {

		case "engine":
			return doStartEngine()

		case "gui":
			return doApi("engine.startprocess", "process", "gui")

		case "bidule":
			return doApi("engine.startprocess", "process", "bidule")

		case "resolume":
			return doApi("engine.startprocess", "process", "resolume")

		case "keykit":
			return doApi("engine.startprocess", "process", "keykit")

		case "", "all":
			s1 := doStartEngine()
			s1 += "\n" + doApi("engine.startprocess", "process", "gui")
			s1 += "\n" + doApi("engine.startprocess", "process", "bidule")
			s1 += "\n" + doApi("engine.startprocess", "process", "resolume")
			s1 += "\n" + doApi("engine.startprocess", "process", "keykit")
			return s1

		default:
			return usage()
		}

	case "stop":

		if !engine.IsEngineRunning() {
			return "Engine is not running."
		}

		switch arg1 {

		case "", "all":
			s1 := doApi("engine.stopprocess", "process", "gui")
			s1 += "\n" + doApi("engine.stopprocess", "process", "bidule")
			s1 += "\n" + doApi("engine.stopprocess", "process", "resolume")
			s1 += "\n" + doApi("engine.stopprocess", "process", "keykit")
			s1 += "\n" + doApi("engine.exit")
			return s1

		case "engine":
			return doApi("engine.exit")

		case "gui":
			return doApi("engine.stopprocess", "process", "gui")

		case "bidule":
			return doApi("engine.stopprocess", "process", "bidule")

		case "resolume":
			return doApi("engine.stopprocess", "process", "resolume")

		case "keykit":
			return doApi("engine.stopprocess", "process", "keykit")

		default:
			return usage()
		}

	case "version":
		return engine.GetPaletteVersion()

	case "sendlogs":
		err = engine.SendLogs()
		if err != nil {
			return err.Error()
		}
		return "Logs have been sent."

	default:
		words := strings.Split(api, ".")
		if len(words) != 2 {
			return "Invalid api format, expecting {plugin}.{api}\n" + usage()
		}
		return doApi(api, args[1:]...)
	}
}

func doApi(api string, apiargs ...string) string {
	resultMap, err := engine.RemoteAPI(api, apiargs...)
	if err != nil {
		return fmt.Sprintf("RemoteAPI: api=%s err=%s\n", api, err)
	}
	return humanReadableApiOutput(resultMap)
}

func doStartEngine() string {

	if engine.IsEngineRunning() {
		return "Engine is already running?"
	}

	err := engine.StartEngine()
	if err != nil {
		return fmt.Sprintf("engine.StartEngine: err=%s", err)
	}
	return "Engine has been started."
}
