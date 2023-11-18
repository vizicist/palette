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
	args := flag.Args()

	// If we're trying to archive the logs, use stdout.
	// don't open any log files, use stdout
	doingArchiveLogs := (len(args) == 1 && args[0] == "archivelogs")
	if doingArchiveLogs {
		engine.InitLog("") // "" == "stdout"
	} else {
		engine.InitLog("palette")
	}

	engine.InitMisc()
	engine.InitEngine()

	engine.LogInfo("Palette InitLog", "args", args)

	apiout, err := CliCommand(args)
	if err != nil {
		os.Stdout.WriteString("Error: " + err.Error() + "\n")
		engine.LogError(err)
	} else {
		os.Stdout.WriteString(engine.HumanReadableApiOutput(apiout))
	}
}

func usage() string {
	return `Invalid usage, expecting:
	palette start [ monitor, engine, gui, bidule, resolume, mmtt ]
	palette kill [ all, monitor, engine, gui, bidule, resolume, mmtt]
	palette status
	palette version
	palette taillog
	palette summarize {logfile}
	palette {category}.{api} [ {argname} {argvalue} ] ...
	`
}

/*
func processStatus(process string) string {
	if engine.IsRunning(process) {
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
		return nil, engine.SummarizeLog(arg1)

	case "taillog", "logtail":
		logpath := engine.LogFilePath("engine.log")
		t, err := tail.TailFile(logpath, tail.Config{Follow: true})
		engine.LogIfError(err)
		for line := range t.Lines {
			fmt.Println(line.Text)
		}
		return nil, nil

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
			// the engine will restart any processes specified in engine.process.*.
			for _, process := range engine.ProcessList() {
				if arg1 == process {
					param := "engine.process." + arg1
					return engine.EngineRemoteApi("engine.set", "name", param, "value", "true")
				}
			}
			return nil, fmt.Errorf("process %s is disabled or unknown", arg1)
		}

	case "kill", "stop":

		switch arg1 {

		case "", "all":
			engine.LogInfo("Palette kill is killing everything including monitor.")
			engine.KillExecutable(engine.MonitorExe)
			engine.KillAllExceptMonitor()
			return nil, nil

		case "monitor":
			engine.KillExecutable(engine.MonitorExe)
			return nil, nil

		case "engine":
			// Don't use engine.exit API, just kill it
			engine.KillExecutable(engine.EngineExe)
			return nil, nil

		default:
			// Individual processes are stopped by setting engine.process.* to false.	
			// If the engine isn't running, this will fail.  Use stop all as last resort.
			param := "engine.process." + arg1
			return engine.EngineRemoteApi("engine.set", "name", param, "value", "false")
		}

	case "version":
		s := engine.GetPaletteVersion()
		return map[string]string{"result": s}, nil

	case "align":
		return engine.MmttApi("align_start")

	case "archivelogs":
		// Make sure nothing is running.
		statusOut, nrunning := StatusOutput()
		if nrunning > 0 {
			return nil, fmt.Errorf("cannot archive logs while processes are running: %s", statusOut)
		}
		return nil, engine.ArchiveLogs()

	case "test":
		return engine.EngineRemoteApi("quadpro.test", "ntimes", "40")

	case "obs":
		engine.LogInfo("palette: obs command")
		err := engine.ObsCommand(arg1)
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		// return map[string]string{"result": ""}, nil
		return nil, nil

	default:
		if len(words) < 2 {
			return nil, fmt.Errorf("unrecognized command (%s), expected usage:\n%s", api, usage())
		} else if len(words) > 2 {
			return nil, fmt.Errorf("invalid api format, expecting {plugin}.{api}\n" + usage())
		}
		return engine.EngineRemoteApi(api, args[1:]...)
	}
}

func StatusOutput() (statusOut string, numRunning int) {
	s := ""
	nrunning := 0
	if engine.MonitorIsRunning() {
		s += "Monitor is running.\n"
		nrunning++
	}

	if engine.IsRunning("engine") {
		s += "Engine is running.\n"
		nrunning++
	}

	if engine.IsRunning("gui") {
		s += "GUI is running.\n"
		nrunning++
	}

	if engine.IsRunning("bidule") {
		s += "Bidule is running.\n"
		nrunning++
	}

	if engine.IsRunning("obs") {
		s += "OBS is running.\n"
		nrunning++
	}

	if engine.IsRunning("chat") {
		s += "Chat monitor is running.\n"
		nrunning++
	}

	if engine.IsRunning("resolume") {
		s += "Resolume is running.\n"
		nrunning++
	}

	b, _ := engine.GetParamBool("engine.keykitrun")
	if b {
		if engine.IsRunning("keykit") {
			s += "Keykit is running.\n"
			nrunning++
		}
	}

	mmtt, _ := engine.GetParam("engine.mmtt")
	if mmtt != "" {
		if engine.IsRunning("mmtt") {
			s += "MMTT is running.\n"
			nrunning++
		}
	}
	return s, nrunning
}

func doStartEngine() error {
	if engine.IsRunning("engine") {
		return fmt.Errorf("engine is already running")
	}
	fullexe := filepath.Join(engine.PaletteDir(), "bin", engine.EngineExe)
	engine.LogInfo("palette: starting engine", "EngineExe", engine.EngineExe)
	return engine.StartExecutableLogOutput("engine", fullexe)
}

func doStartMonitor() error {
	if engine.MonitorIsRunning() {
		return fmt.Errorf("monitor is already running")
	}
	// palette_monitor.exe will restart the engine,
	// which then starts whatever engine.process.* specifies.
	engine.LogInfo("palette: starting monitor", "MonitorExe", engine.MonitorExe)
	fullexe := filepath.Join(engine.PaletteDir(), "bin", engine.MonitorExe)
	return engine.StartExecutableLogOutput("monitor", fullexe)
}

