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
	"strings"
	"syscall"
	"time"

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

func RemoteAPI(api string, args ...string) string {

	if len(args)%2 != 0 {
		return "RemoteAPI: odd nnumber of args, should be even"
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

// If it's not a layer, it's a button.
func CliCommand(args []string) string {

	var err error

	if len(args) == 0 {
		return usage()
	}

	const engineexe = "palette_engine.exe"

	api := args[0]

	switch api {

	case "status":
		if engine.IsRunningExecutable(engineexe) {
			return "Engine is running."
		} else {
			return "Engine is stopped."
		}

	case "start":

		if engine.IsRunningExecutable(engineexe) {
			return "Engine is already running?"
		}

		// Kill any currently-running engine.
		// The other processes will be killed by
		// the engine when it starts up.
		engine.KillExecutable(engineexe)

		// Start the engine (which also starts up other processes)
		fullexe := filepath.Join(engine.PaletteDir(), "bin", engineexe)
		err = engine.StartExecutableLogOutput("engine", fullexe, true, "")
		if err != nil {
			return fmt.Sprintf("Engine not started: err=%s", err)
		}
		return "Engine has been started."

	case "stop":
		// kill ourselves
		engine.KillExecutable(engineexe)
		return "Engine has been stopped."

	case "version":
		return engine.GetPaletteVersion()

	case "sendlogs":
		err = engine.SendLogs()
		if err != nil {
			return err.Error()
		}
		return "Logs have been sent."

	case "test":
		var ntimes = 10 // default
		var dt = 100 * time.Millisecond
		if len(args) > 1 {
			ntimes, err = strconv.Atoi(args[1])
			if err != nil {
				return err.Error()
			}
		}
		if len(args) > 2 {
			dt, err = time.ParseDuration(args[2])
			if err != nil {
				return err.Error()
			}
		}
		doTest(ntimes, dt)
		return "Test is complete."

	default:
		words := strings.Split(api, ".")
		if len(words) != 2 {
			return "Invalid api format, expecting {plugin}.{api}"
		}
		return RemoteAPI(api, args[1:]...)
	}
}

func doTest(ntimes int, dt time.Duration) {
	source := "test.0"
	for n := 0; n < ntimes; n++ {
		if n > 0 {
			time.Sleep(dt)
		}
		layer := string("ABCD"[rand.Int()%4])
		RemoteAPI("event",
			"layer", layer,
			"source", source,
			"event", "cursor_down",
			"x", fmt.Sprintf("%f", rand.Float32()),
			"y", fmt.Sprintf("%f", rand.Float32()),
			"z", fmt.Sprintf("%f", rand.Float32()),
		)
		time.Sleep(dt)
		RemoteAPI("event",
			"layer", layer,
			"source", source,
			"event", "cursor_up",
			"x", fmt.Sprintf("%f", rand.Float32()),
			"y", fmt.Sprintf("%f", rand.Float32()),
			"z", fmt.Sprintf("%f", rand.Float32()),
		)
	}
}
