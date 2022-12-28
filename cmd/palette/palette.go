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

	out := CliCommand(flag.Args())
	os.Stdout.WriteString(out)
}

// Usage for controlling the "ppro" (PalettePro) plugin
func usage() string {
	return `Commands:
	palette start
	palette stop
	palette version
	palette {plugin}.{api} [ {argname} {argvalue} ] ...
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

func engineStop(stopOrStopAll string) string {

	if !engine.IsEngineRunning() {
		return "Engine is not running."
	}

	_, err := engine.RemoteAPI("engine." + stopOrStopAll)
	if err != nil {
		return fmt.Sprintf("RemoteAPI: err=%s\n", err)
	}

	if stopOrStopAll == "stopall" {
		return "Engine and Plugins are being stopped."
	} else {
		return "Engine is being stopped."
	}
}

// If it's not a layer, it's a button.
func CliCommand(args []string) string {

	var err error

	if len(args) == 0 {
		return usage()
	}

	api := args[0]

	switch api {

	case "status":
		if engine.IsEngineRunning() {
			return "Engine is running."
		} else {
			return "Engine is stopped."
		}

	case "start", "engine.start":
		return doStart()

	case "startall", "engine.startall":
		// Someday it should make a difference, so "start" doesn't start the plugins.
		return doStart()

	case "stop", "engine.stop":
		return engineStop("stop")

	case "stopall", "engine.stopall":
		return engineStop("stopall")

	case "version":
		return engine.GetPaletteVersion()

	case "sendlogs":
		err = engine.SendLogs()
		if err != nil {
			return err.Error()
		}
		return "Logs have been sent."

	case "gui":
		return doApi("ppro.startprocess", "process", "gui")

	default:
		words := strings.Split(api, ".")
		if len(words) != 2 {
			return "Invalid api format, expecting {plugin}.{api}"
		}
		return doApi(api, args[1:]...)
	}
}

func doApi(api string, apiargs ...string) string {
	resultMap, err := engine.RemoteAPI(api, apiargs...)
	if err != nil {
		return fmt.Sprintf("RemoteAPI: err=%s\n", err)
	}
	return humanReadableApiOutput(resultMap)
}

func doStart() string {

	if engine.IsEngineRunning() {
		return "Engine is already running?"
	}

	err := engine.StartEngine()
	if err != nil {
		return fmt.Sprintf("engine.StartEngine: err=%s", err)
	}
	return "Engine has been started."
}
