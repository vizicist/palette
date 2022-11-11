package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/vizicist/palette/engine"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	// Does anything need to be initia
	flag.Parse()

	engine.InitLog("palette")

	playerPtr := flag.String("player", "*", "Region name or *")

	out := CliCommand(*playerPtr, flag.Args())
	os.Stdout.WriteString(out)
}

func usage() string {
	return `Commands:
    palette start [process]
    palette stop [process]
    palette activate
    palette sendlogs
    palette list [{category}]
    palette [-player={player}] load {category}.{preset}
    palette [-player {player}] save {category}.{preset}
    palette [-player {player}] set {category}.{parameter} [{value}]
    palette [-player {player}] get {category}.{parameter}
    palette status
	palette version
    palette api {api} {args}
	
Regions:
    A, B, C, D, *

Events:
    midiin, midiout, cursor

Categories:
    quad, snap, sound, visual, effect`
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
	return result
}

func RemoteAPI(api string, args ...string) string {

	if len(args)%2 != 0 {
		return "RemoteAPI: odd nnumber of args, should be even\n"
	}
	apijson := "\"api\": \"" + api + "\""
	for n := range args {
		if n%2 == 0 {
			apijson = apijson + ",\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
	}
	resultMap := RemoteAPIRaw(apijson)
	// result should be a json string
	return humanReadableApiOutput(resultMap)
}

func RemoteAPIRaw(args string) map[string]string {
	url := fmt.Sprintf("http://127.0.0.1:%d/api", engine.HTTPPort)
	postBody := []byte(args)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		output := make(map[string]string)
		output["error"] = fmt.Sprintf("RemoteAPIRaw: Post err=%s", err)
		return output
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]string{"error": fmt.Sprintf("RemoteAPIRaw: ReadAll err=%s", err)}
	}
	output, err := engine.StringMap(string(body))
	if err != nil {
		return map[string]string{"error": fmt.Sprintf("RemoteAPIRaw: unable to interpret output, err=%s", err)}
	}
	return output
}

// If it's not a player, it's a button.
func CliCommand(player string, args []string) string {

	if len(args) == 0 {
		return usage()
	}

	const engineexe = "palette_engine.exe"

	api := args[0]
	switch api {

	case "load", "save":
		if len(args) < 2 {
			return "Insufficient arguments"
		}
		return RemoteAPI(api, "player", player, "preset", args[1])

	case "get":
		if len(args) < 2 {
			return "Insufficient arguments"
		}
		return RemoteAPI("get", "player", player, "name", args[1])

	case "set":
		if len(args) < 3 {
			return "Insufficient arguments"
		}
		return RemoteAPI("set", "player", args[1], "name", args[1], "value", args[2])

	case "preset.list":
		category := "*"
		if len(args) > 1 {
			category = args[1]
		}
		return RemoteAPI("preset.list", "category", category)

	case "sendlogs":
		return RemoteAPI("global.sendlogs")

	case "status", "tasks":
		return engine.ProcessStatus()

	case "version":
		return engine.GetPaletteVersion()

	case "activate":
		return RemoteAPI("activate")

	case "nextalive":
		return RemoteAPI("nextalive")

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
			return RemoteAPI("start", "process", process)
		}

	case "stop":
		process := "engine"
		if len(args) > 1 {
			process = args[1]
		}
		if process == "all" {
			engine.StopRunning("all")
		} else if process == "engine" {
			// first stop everything else, unless killonstartup is false
			if engine.ConfigBoolWithDefault("killonstartup", true) {
				engine.StopRunning("all")
			}
			// then kill ourselves
			engine.KillExecutable(engineexe)
		} else {
			return RemoteAPI("stop", "process", process)
		}

	case "api":
		if len(args) < 2 {
			return "Insufficient arguments"
		}
		return RemoteAPI(args[1])

	case "test":
		ntimes := 10
		dt := 100 * time.Millisecond
		source := "test.0"
		var err error
		if len(args) > 1 {
			ntimes, err = strconv.Atoi(args[1])
			if err != nil {
				return err.Error()
			}
			if len(args) > 2 {
				dt, err = time.ParseDuration(args[2])
				if err != nil {
					return err.Error()
				}
			}
		}
		for n := 0; n < ntimes; n++ {
			if n > 0 {
				time.Sleep(dt)
			}
			player := string("ABCD"[rand.Int()%4])
			RemoteAPI("event",
				"player", player,
				"source", source,
				"event", "cursor_down",
				"x", fmt.Sprintf("%f", rand.Float32()),
				"y", fmt.Sprintf("%f", rand.Float32()),
				"z", fmt.Sprintf("%f", rand.Float32()),
			)
			time.Sleep(dt)
			RemoteAPI("event",
				"player", player,
				"source", source,
				"event", "cursor_up",
				"x", fmt.Sprintf("%f", rand.Float32()),
				"y", fmt.Sprintf("%f", rand.Float32()),
				"z", fmt.Sprintf("%f", rand.Float32()),
			)
		}

	default:
		// NEW retmap
		return fmt.Sprintf("Unrecognized command: %s", args[0])
	}
	return ""
}
