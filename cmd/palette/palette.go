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
	"github.com/vizicist/palette/hostwin"
	"github.com/vizicist/palette/kit"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	flag.Parse()
	args := flag.Args()

	// If we're trying to archive the logs, use stdout.
	// don't open any log files, use stdout
	doingArchiveLogs := (len(args) == 1 && args[0] == "archivelogs")
	logname := ""
	if ! doingArchiveLogs {
		logname = "palette"
	}
	host := hostwin.NewHost(logname)

	host.LogInfo("Palette", "args", args)

	apiout, err := CliCommand(args)
	if err != nil {
		os.Stdout.WriteString("Error: " + err.Error() + "\n")
		host.LogError(err)
	} else {
		os.Stdout.WriteString(hostwin.HumanReadableApiOutput(apiout))
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

	// Warn about extra unexpected arguments
	switch api {
	case "start", "kill", "stop", "summarize":
		// okay
	default:
		// extra args are allowed on api.method usage
		if len(words) != 2 && len(args) > 1 {
			return nil, fmt.Errorf(usage())
		}
	}

	switch api {

	case "summarize":
		return nil, hostwin.SummarizeLog(arg1)

	case "taillog", "logtail":
		logpath := hostwin.LogFilePath("engine.log")
		t, err := tail.TailFile(logpath, tail.Config{Follow: true})
		hostwin.LogIfError(err)
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
			if hostwin.MonitorIsRunning() {
				return nil, fmt.Errorf("monitor is already running")
			}

			// palette_monitor.exe will restart the engine,
			// which then starts whatever engine.autostart specifies.

			hostwin.LogInfo("palette: starting monitor", "MonitorExe", hostwin.MonitorExe)
			fullexe := filepath.Join(hostwin.PaletteDir(), "bin", hostwin.MonitorExe)
			return nil, hostwin.StartExecutableLogOutput("monitor", fullexe)

		case "engine":
			return nil, doStartEngine()

		default:
			// If it exists in the ProcessList...
			for _, process := range hostwin.ProcessList() {
				if arg1 == process {
					return hostwin.EngineApi("engine.startprocess", "process", arg1)
				}
			}
			return nil, fmt.Errorf("process %s is disabled or unknown", arg1)
		}

	case "kill", "stop":

		switch arg1 {

		case "", "all":
			hostwin.LogInfo("Palette kill is killing everything including monitor.")
			hostwin.KillExecutable(hostwin.MonitorExe)
			hostwin.KillAllExceptMonitor()
			return nil, nil

		case "monitor":
			hostwin.KillExecutable(hostwin.MonitorExe)
			return nil, nil

		case "engine":
			// Don't use engine.exit API, just kill it
			hostwin.KillExecutable(hostwin.EngineExe)
			return nil, nil

		default:
			// If it exists in the ProcessList...
			pi, err := hostwin.TheProcessManager.GetProcessInfo(arg1)
			if err != nil {
				return nil, err
			}
			hostwin.KillExecutable(pi.Exe)
			return nil, nil
		}

	case "version":
		s := hostwin.GetPaletteVersion()
		return map[string]string{"result": s}, nil

	case "align":
		return hostwin.MmttApi("align_start")

	case "archivelogs":
		// Make sure nothing is running.
		statusOut, nrunning := StatusOutput()
		if nrunning > 0 {
			return nil, fmt.Errorf("cannot archive logs while processes are running:\n%s\n", statusOut)
		}
		return nil, hostwin.ArchiveLogs()

	case "test":
		return hostwin.EngineApi("quadpro.test", "ntimes", "40")

	default:
		if len(words) < 2 {
			return nil, fmt.Errorf("unrecognized command (%s), expected usage:\n%s", api, usage())
		} else if len(words) > 2 {
			return nil, fmt.Errorf("invalid api format, expecting {plugin}.{api}" + usage())
		}
		return hostwin.EngineApi(api, args[1:]...)
	}
}

func StatusOutput() (statusOut string, numRunning int) {
	s := ""
	nrunning := 0
	if hostwin.MonitorIsRunning() {
		s += "Monitor is running.\n"
		nrunning++
	}

	if hostwin.IsRunning("engine") {
		s += "Engine is running.\n"
		nrunning++
	}

	if hostwin.IsRunning("gui") {
		s += "GUI is running.\n"
		nrunning++
	}

	if hostwin.IsRunning("bidule") {
		s += "Bidule is running.\n"
		nrunning++
	}

	if hostwin.IsRunning("resolume") {
		s += "Resolume is running.\n"
		nrunning++
	}

	mmtt, _ := kit.TheHost.GetParam("engine.mmtt")
	if mmtt != "" {
		if hostwin.IsRunning("mmtt") {
			s += "MMTT is running.\n"
			nrunning++
		}
	}
	return s, nrunning
}

func doStartEngine() error {

	if hostwin.IsRunning("engine") {
		return fmt.Errorf("engine is already running")
	}

	fullexe := filepath.Join(hostwin.PaletteDir(), "bin", hostwin.EngineExe)
	err := hostwin.StartExecutableLogOutput("engine", fullexe)
	if err != nil {
		hostwin.LogIfError(err)
		return err
	}
	return nil
}
