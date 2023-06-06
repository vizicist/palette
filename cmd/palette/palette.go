package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nxadm/tail"
	"github.com/vizicist/palette/engine"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	flag.Parse()

	engine.InitLog("palette")
	engine.InitMisc()
	engine.InitEngine()

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

func statusString(process string) string {
	if engine.IsRunning(process) {
		return "running"
	}
	return "not running"
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

	case "taillog":
		logpath := engine.LogFilePath("engine.log")
		t, err := tail.TailFile(logpath, tail.Config{Follow: true})
		engine.LogIfError(err)
		for line := range t.Lines {
			fmt.Println(line.Text)
		}
		return ""

	case "status":
		s := ""
		s += "Engine is " + statusString("engine") + ".\n"
		s += "GUI is " + statusString("gui") + ".\n"
		s += "Bidule is " + statusString("bidule") + ".\n"
		s += "Resolume is " + statusString("resolume") + ".\n"
		b, _ := engine.GetParamBool("engine.keykit")
		if b {
			s += "Keykit is " + statusString("keykit") + ".\n"
		}
		mmtt, _ := engine.GetParam("engine.mmtt")
		if mmtt != "" {
			s += "MMTT is " + statusString("mmtt") + ".\n"
		}
		return s

	case "start":

		switch arg1 {

		case "", "engine":
			return doStartEngine()

		case "all":
			s := doStartEngine()
			for _, process := range engine.ProcessList() {
				s += "\n" + doApi("engine.startprocess", "process", process)
			}
			return s

		default:
			// If it exists in the ProcessList...
			for _, process := range engine.ProcessList() {
				if arg1 == process {
					return doApi("engine.startprocess", "process", arg1)
				}
			}
			return fmt.Sprintf("Process %s is disabled or unknown.\n", arg1)
		}

	case "stop":

		if !engine.IsRunning("engine") {
			return "Engine is not running."
		}

		switch arg1 {

		case "all":
			sep := ""
			s := ""
			for _, process := range engine.ProcessList() {
				s += sep + doApi("engine.stopprocess", "process", process)
				sep = "\n"
			}
			s += sep + doApi("engine.exit")
			return s

		case "", "engine":
			return doApi("engine.exit")

		default:
			// If it exists in the ProcessList...
			for _, process := range engine.ProcessList() {
				if arg1 == process {
					return doApi("engine.stopprocess", "process", arg1)
				}
			}
			return fmt.Sprintf("Process %s is disabled or unknown.\n", arg1)
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

	if engine.IsRunning("engine") {
		return "Engine is already running?"
	}

	err := engine.StartEngine()
	if err != nil {
		return fmt.Sprintf("engine.StartEngine: err=%s", err)
	}
	return "Engine has been started."
}
