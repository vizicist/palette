package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vizicist/palette/kit"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	flag.Parse()
	args := flag.Args()

	// If we're doing any log commands, use stdout
	doingLogs := (len(args) > 0 && args[0] == "logs")
	if doingLogs {
		kit.InitLog("")
	} else {
		kit.InitLog("palette")
	}

	engine.InitMisc()
	// kit.InitEngine()

	kit.LogInfo("Palette InitLog", "args", args)

	apiout, err := CliCommand(args)
	if err != nil {
		os.Stdout.WriteString("Error: " + err.Error() + "\n")
		kit.LogError(err)
	} else {
		os.Stdout.WriteString(kit.HumanReadableApiOutput(apiout))
	}
}

func usage() string {
	return `Invalid usage, expecting:
	palette start [ monitor, engine, gui, bidule, resolume, mmtt ]
	palette kill [ all, monitor, engine, gui, bidule, resolume, mmtt]
	palette status
	palette version
	palette logs [ archive, clear ]
	palette summarize {logfile}
	palette {category}.{api} [ {argname} {argvalue} ] ...
	`
}

/*
func processStatus(process string) string {
	if kit.IsRunning(process) {
		return "running"
	}
	return "not running"
}
*/

func CliCommand(args []string) (map[string]string, error) {

	if len(args) == 0 {
		return nil, fmt.Errorf(usage())
	}

	api := args[0]
	var arg1 string
	if len(args) > 1 {
		arg1 = args[1]
	}

	words := strings.Split(api, ".")

	switch api {

	case "summarize":
		return nil, kit.SummarizeLog(arg1)

	case "status":
		s, nrunning := StatusOutput()
		if nrunning == 0 {
			s = "No palette processes are running."
		}
		out := map[string]string{"result": s}
		return out, nil

	case "start":

		switch arg1 {

		case "", "monitor":
			return nil, doStartMonitor()

		case "engine":
			return nil, doStartEngine()

		default:
			// Only the monitor and engine are started directly by palette.
			// The monitor will restart the engine if it dies, and
			// the engine will restart any processes specified in global.process.*.
			for _, process := range kit.ProcessList() {
				if arg1 == process {
					param := "global.process." + arg1
					return kit.EngineRemoteApi("global.set", "name", param, "value", "true")
				}
			}
			return nil, fmt.Errorf("process %s is disabled or unknown", arg1)
		}

	case "kill", "stop":

		switch arg1 {

		case "", "all":
			kit.LogInfo("Palette kill is killing everything including monitor.")
			kit.KillExecutable(kit.MonitorExe)
			kit.KillAllExceptMonitor()
			return nil, nil

		case "monitor":
			kit.KillExecutable(kit.MonitorExe)
			return nil, nil

		case "engine":
			// Don't use kit.exit API, just kill it
			kit.KillExecutable(kit.EngineExe)
			return nil, nil

		default:
			// Individual processes are stopped by setting global.process.* to false.
			// If the engine isn't running, this will fail.  Use stop all as last resort.
			param := "global.process." + arg1
			return kit.EngineRemoteApi("global.set", "name", param, "value", "false")
		}

	case "version":
		s := kit.GetPaletteVersion()
		return map[string]string{"result": s}, nil

	case "align":
		return kit.MmttApi("align_start")

	case "logs":
		switch arg1 {
		case "archive":
			// Make sure nothing is running.
			statusOut, nrunning := StatusOutput()
			if nrunning > 0 {
				return nil, fmt.Errorf("cannot archive logs while these processes are running:\n%s", statusOut)
			}
			return nil, kit.ArchiveLogs()
		case "clear":
			return nil, kit.ClearLogs()
			// case "tail":
			// 	return nil, kit.TailLogs()
		}
		return nil, fmt.Errorf("invalid logs command: %s", arg1)

	case "test":
		return kit.EngineRemoteApi("quadpro.test", "ntimes", "40")

	case "obs":
		kit.LogInfo("palette: obs command")
		err := kit.ObsCommand(arg1)
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		// return map[string]string{"result": ""}, nil
		return nil, nil

	case "nats":
		kit.LogInfo("palette: nats command")
		if len(args) < 2 {
			return nil, fmt.Errorf("nats command missing argument")
		}
		result, err := kit.NatsApi(args[1])
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		} else {
			return map[string]string{"result": result}, nil
		}

	default:
		if len(words) < 2 {
			return nil, fmt.Errorf("unrecognized command (%s), expected usage:\n%s", api, usage())
		} else if len(words) > 2 {
			return nil, fmt.Errorf("invalid api format, expecting {plugin}.{api}\n" + usage())
		}
		return kit.EngineRemoteApi(api, args[1:]...)
	}
}

func StatusOutput() (statusOut string, numRunning int) {
	s := ""
	nrunning := 0
	if kit.MonitorIsRunning() {
		s += "Monitor is running.\n"
		nrunning++
	}

	if kit.IsRunning("engine") {
		s += "Engine is running.\n"
		nrunning++
	}

	if kit.IsRunning("gui") {
		s += "GUI is running.\n"
		nrunning++
	}

	if kit.IsRunning("bidule") {
		s += "Bidule is running.\n"
		nrunning++
	}

	if kit.IsRunning("obs") {
		s += "OBS is running.\n"
		nrunning++
	}

	if kit.IsRunning("chat") {
		s += "Chat monitor is running.\n"
		nrunning++
	}

	if kit.IsRunning("resolume") {
		s += "Resolume is running.\n"
		nrunning++
	}

	b, _ := kit.GetParamBool("global.keykitrun")
	if b {
		if kit.IsRunning("keykit") {
			s += "Keykit is running.\n"
			nrunning++
		}
	}

	mmtt, _ := kit.GetParam("global.mmtt")
	if mmtt != "" {
		if kit.IsRunning("mmtt") {
			s += "MMTT is running.\n"
			nrunning++
		}
	}
	return s, nrunning
}

func doStartEngine() error {
	if kit.IsRunning("engine") {
		return fmt.Errorf("engine is already running")
	}
	fullexe := filepath.Join(kit.PaletteDir(), "bin", kit.EngineExe)
	kit.LogInfo("palette: starting engine", "EngineExe", kit.EngineExe)
	return kit.StartExecutableLogOutput("engine", fullexe)
}

func doStartMonitor() error {
	if kit.MonitorIsRunning() {
		return fmt.Errorf("monitor is already running")
	}
	// palette_monitor.exe will restart the engine,
	// which then starts whatever global.process.* specifies.
	kit.LogInfo("palette: starting monitor", "MonitorExe", kit.MonitorExe)
	fullexe := filepath.Join(kit.PaletteDir(), "bin", kit.MonitorExe)
	return kit.StartExecutableLogOutput("monitor", fullexe)
}
