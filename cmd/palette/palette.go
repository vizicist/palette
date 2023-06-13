package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
	palette start
	palette stop
	palette status
	palette version
	palette logtail
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
		result = "OK\n"
	}

	return result
}

func processStatus(process string) string {
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

	case "taillog", "logtail":
		if arg1 != "" {
			return usage()
		}
		logpath := engine.LogFilePath("engine.log")
		t, err := tail.TailFile(logpath, tail.Config{Follow: true})
		engine.LogIfError(err)
		for line := range t.Lines {
			fmt.Println(line.Text)
		}
		return ""

	case "status":
		if arg1 != "" {
			return usage()
		}
		s := ""
		s += "Monitor is " + monitorStatus() + ".\n"
		s += "Engine is " + processStatus("engine") + ".\n"
		s += "GUI is " + processStatus("gui") + ".\n"
		s += "Bidule is " + processStatus("bidule") + ".\n"
		s += "Resolume is " + processStatus("resolume") + ".\n"
		b, _ := engine.GetParamBool("engine.keykitrun")
		if b {
			s += "Keykit is " + processStatus("keykit") + ".\n"
		}
		mmtt, _ := engine.GetParam("engine.mmtt")
		if mmtt != "" {
			s += "MMTT is " + processStatus("mmtt") + ".\n"
		}
		return s

	case "start":

		switch arg1 {

		case "":
			// palette_monitor.exe will restart the engine,
			// which then starts whatever engine.autostart specifies.
			return doStartMonitor()

		case "engine":
			return doStartEngine()

		default:
			// If it exists in the ProcessList...
			for _, process := range engine.ProcessList() {
				if arg1 == process {
					return doApi("engine.startprocess", "process", arg1)
				}
			}
			return fmt.Sprintf("Process %s is disabled or unknown.\n", arg1)
		}

	case "kill":
		if arg1 != "" {
			return usage()
		}
		engine.LogInfo("Palette stop is killing everything, including monitor.")
		engine.KillAllExceptMonitor()
		engine.KillMonitor()
		return "OK\n"

	case "stop":

		switch arg1 {

		case "":
			engine.LogInfo("Palette stop is killing everything including monitor.")
			doApi("engine.showclip", "clipnum", "3")
			time.Sleep(time.Second * 2)
			engine.KillAllExceptMonitor()
			engine.KillMonitor()
			return "OK\n"

		case "engine":
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
		if arg1 != "" {
			return usage()
		}
		return engine.GetPaletteVersion()

	case "restart":
		if arg1 != "" {
			return usage()
		}
		engine.KillAllExceptMonitor()
		engine.KillMonitor()
		time.Sleep(time.Second * 2)
		return doStartMonitor()

	case "sendlogs":
		if arg1 != "" {
			return usage()
		}
		err = engine.SendLogs()
		if err != nil {
			return err.Error()
		}
		return "Logs have been sent."

	case "test":
		return doApi("quadpro.test", "ntimes", "40")

	default:
		words := strings.Split(api, ".")
		if len(words) != 2 {
			return "Invalid api format, expecting {plugin}.{api}\n" + usage()
		}
		return doApi(api, args[1:]...)
	}
}

func monitorStatus() string {
	if engine.IsRunningExecutable(engine.MonitorExe) {
		return "running"
	} else {
		return "not running"
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

	fullexe := filepath.Join(engine.PaletteDir(), "bin", engine.EngineExe)
	err := engine.StartExecutableLogOutput("engine", fullexe)

	if err != nil {
		return fmt.Sprintf("engine.StartEngine: err=%s", err)
	}
	return "Engine has been started."
}

func doStartMonitor() string {

	if engine.IsRunningExecutable(engine.MonitorExe) {
		return "Monitor is already running?"
	}

	engine.LogInfo("doStartMonitor: starting monitor")
	fullexe := filepath.Join(engine.PaletteDir(), "bin", engine.MonitorExe)
	err := engine.StartExecutableLogOutput("monitor", fullexe)
	if err != nil {
		engine.LogIfError(err)
		return fmt.Sprintf("engine.StartMonitor: err=%s", err)
	}
	return "Monitor has been started."
}
