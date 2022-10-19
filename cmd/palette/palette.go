package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/vizicist/palette/engine"
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
	return `Commands:
    palette start [process]
    palette stop [process]
    palette activate
    palette sendlogs
    palette list [{category}]
    palette [-region={region}] load {category}.{preset}
    palette [-region {region}] save {category}.{preset}
    palette [-region {region}] set {category}.{parameter} [{value}]
    palette [-region {region}] get {category}.{parameter}
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
	output, err := RemoteAPIRaw(apijson)
	// If err is non-nill, interpretApiOutput will include it
	return interpretApiOutput(output, err)
}

func RemoteAPIRaw(args string) (output string, err error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api", engine.HTTPPort)
	postBody := []byte(args)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return "", fmt.Errorf("RemoteAPIRaw: Post err=%s", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("RemoteAPIRaw: ReadAll err=%s", err)
	}
	arr, err := engine.StringMap(string(body))
	if err != nil {
		return "", fmt.Errorf("RemoteAPIRaw: bad format of api response body")
	}
	errstr, eok := arr["error"]
	if eok {
		return "", fmt.Errorf("RemoteAPIRaw: err=%s", errstr)
	}
	resultstr, rok := arr["result"]
	if !rok {
		return "", fmt.Errorf("RemoteAPIRaw: no result value in api response?")
	}
	return resultstr, nil

	/*
		resp, err := http.Get(url)
		if err != nil {
			return "", fmt.Errorf("PaletteAPI: err=%s", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("PaletteAPI: ReadAll err=%s", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("PaletteAPI: http status %d", resp.StatusCode)
		}
		fmt.Printf("PaletteAPI: statuscode=%d, body=%s\n", resp.StatusCode, resp.Body)
		return string(body), nil
	*/
}

// If it's not a region, it's a button.
func CliCommand(region string, args []string) string {

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
		return RemoteAPI("api", api, "region", region, "preset", args[1])

	case "get":
		if len(args) < 2 {
			return "Insufficient arguments"
		}
		return RemoteAPI("get", "region", region, "name", args[1])

	case "set":
		if len(args) < 3 {
			return "Insufficient arguments"
		}
		return RemoteAPI("set", "region", args[1], "name", args[1], "value", args[2])

	case "list":
		category := "*"
		if len(args) > 1 {
			category = args[1]
		}
		return RemoteAPI("list", "category", category)

	case "sendlogs":
		return RemoteAPI("global.sendlogs")

	case "status", "tasks":
		return engine.ProcessStatus()

	case "version":
		return engine.PaletteVersion()

	case "activate":
		return RemoteAPI("activate")

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
			engine.TheRouter().StopRunning("all")
		} else if process == "engine" {
			// first stop everything else, unless killonstartup is false
			if engine.ConfigBoolWithDefault("killonstartup", true) {
				engine.TheRouter().StopRunning("all")
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
		return RemoteAPI("api", args[1])

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
			region := string("ABCD"[rand.Int()%4])
			RemoteAPI("event",
				"region", region,
				"source", source,
				"event", "cursor_down",
				"x", fmt.Sprintf("%f", rand.Float32()),
				"y", fmt.Sprintf("%f", rand.Float32()),
				"z", fmt.Sprintf("%f", rand.Float32()),
			)
			time.Sleep(dt)
			RemoteAPI("event",
				"region", region,
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
