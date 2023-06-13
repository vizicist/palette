package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

	apiout, err := CliCommand(flag.Args())
	if err != nil {
		os.Stdout.WriteString("Error: " + err.Error() + "\n")
	} else {
		os.Stdout.WriteString(engine.HumanReadableApiOutput(apiout))
	}
}

func usage() string {
	return `Invalid usage, expecting:
	palette start [ monitor, engine, gui, bidule, resolume, mmtt ]
	palette kill [ monitor, engine, gui, bidule, resolume, mmtt ]
	palette status
	palette version
	palette logtail
	palette {category}.{api} [ {argname} {argvalue} ] ...
	`
}

func processStatus(process string) string {
	if engine.IsRunning(process) {
		return "running"
	}
	return "not running"
}

func CliCommand(args []string) (map[string]string, error) {

	if len(args) == 0 {
		return nil, fmt.Errorf(usage())
	}

	api := args[0]
	var arg1 string
	if len(args) > 1 {
		arg1 = args[1]
	}

	// The only apis that can handle arguments...
	switch api {
	case "start", "kill":
		// okay
	default:
		if len(args) > 1 {
			return nil, fmt.Errorf(usage())
		}
	}

	switch api {

	case "taillog", "logtail":
		logpath := engine.LogFilePath("engine.log")
		t, err := tail.TailFile(logpath, tail.Config{Follow: true})
		engine.LogIfError(err)
		for line := range t.Lines {
			fmt.Println(line.Text)
		}
		return nil, nil

	case "status":
		s := ""
		if engine.MonitorIsRunning() {
			s += "Monitor is running.\n"
		} else {
			s += "Monitor is not running.\n"
		}
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
		out := map[string]string{"result": s}
		return out, nil

	case "start":

		switch arg1 {

		case "":
			return nil, fmt.Errorf("palette start requires an additional argument")

		case "monitor":
			if engine.MonitorIsRunning() {
				return nil, fmt.Errorf("monitor is already running")
			}

			// palette_monitor.exe will restart the engine,
			// which then starts whatever engine.autostart specifies.

			engine.LogInfo("palette: starting monitor", "MonitorExe", engine.MonitorExe)
			fullexe := filepath.Join(engine.PaletteDir(), "bin", engine.MonitorExe)
			return nil, engine.StartExecutableLogOutput("monitor", fullexe)

		case "engine":
			return nil, doStartEngine()

		default:
			// If it exists in the ProcessList...
			for _, process := range engine.ProcessList() {
				if arg1 == process {
					return engine.EngineApi("engine.startprocess", "process", arg1)
				}
			}
			return nil, fmt.Errorf("process %s is disabled or unknown", arg1)
		}

	case "kill":

		switch arg1 {

		case "":
			return nil, fmt.Errorf("palette kill requires an additional argument")

		case "all":
			engine.LogInfo("Palette kill is killing everything including monitor.")
			engine.KillExecutable(engine.MonitorExe)
			engine.KillAllExceptMonitor()
			return nil, nil

		case "monitor":
			return nil, engine.KillExecutable(engine.MonitorExe)

		case "engine":
			// Don't use engine.exit API, just kill it
			return nil, engine.KillExecutable(engine.EngineExe)

		default:
			// If it exists in the ProcessList...
			for _, process := range engine.ProcessList() {
				if arg1 == process {
					return engine.EngineApi("engine.stopprocess", "process", arg1)
				}
			}
			return nil, fmt.Errorf("process %s is disabled or unknown", arg1)
		}

	case "version":
		s := engine.GetPaletteVersion()
		return map[string]string{"result": s}, nil

	case "align":
		return engine.MmttApi("realign")

	case "sendlogs":
		return nil, engine.SendLogs()

	case "test":
		return engine.EngineApi("quadpro.test", "ntimes", "40")

	default:
		words := strings.Split(api, ".")
		if len(words) < 2 {
			return nil, fmt.Errorf("unrecognized command (%s), expected usage:\n%s", api, usage())
		} else if len(words) > 2 {
			return nil, fmt.Errorf("invalid api format, expecting {plugin}.{api}" + usage())
		}
		return engine.EngineApi(api, args[1:]...)
	}
}

func doStartEngine() error {

	if engine.IsRunning("engine") {
		return fmt.Errorf("engine is already running")
	}

	fullexe := filepath.Join(engine.PaletteDir(), "bin", engine.EngineExe)
	err := engine.StartExecutableLogOutput("engine", fullexe)
	if err != nil {
		engine.LogIfError(err)
		return err
	}
	return nil
}
