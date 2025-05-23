package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	json "github.com/goccy/go-json"
	"github.com/hypebeast/go-osc/osc"
	"github.com/joho/godotenv"
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

	kit.InitKit()
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
	return `Usage:
	palette start [ {processname} ]
	palette stop [ {processname}]
	palette status
	palette version
	palette env [ set {name} {value} | get {name} ]
	palette logs [ archive, clear ]
	palette summarize {logfile}
	palette osc listen {port@host}
	palette osc send {port@host} {addr} ...
	palette hub [ streams | dumpload ]
	palette get {name}
	palette set {name} {value}
	palette setboot {name} {value}
	palette {category}.{api} [ {argname} {argvalue} ] ...
	`
}

func CliCommand(args []string) (map[string]string, error) {

	if len(args) == 0 {
		return nil, fmt.Errorf("%s", usage())
	}

	api := args[0]
	var arg1 string
	if len(args) > 1 {
		arg1 = args[1]
	}

	words := strings.Split(api, ".")

	switch api {

	case "status":
		s, nrunning := StatusOutput()
		if nrunning == 0 {
			s = "No palette processes are running."
		}
		out := map[string]string{"result": s}
		return out, nil

	case "restart":
		_, err := CliCommand([]string{"stop"})
		if err != nil {
			return nil, err
		}
		_, err = CliCommand([]string{"start"})
		if err != nil {
			return nil, err
		}
		out := map[string]string{"result": "RESTARTED"}
		return out, nil

	// case "api":
	// 	return kit.EngineApi(arg1, args[2:])

	case "osc":

		switch arg1 {

		case "listen":
			if len(args) < 3 {
				return nil, fmt.Errorf("bad osc command (%s), expected usage:\n%s", arg1, usage())
			}
			port, host, err := getPortHost(args[2])
			if err != nil {
				return nil, err
			}
			fmt.Printf("Listening on %d@%s ...\n", port, host)
			ListenAndPrint(host, port)
			return nil, nil

		case "send":
			if len(args) < 3 {
				return nil, fmt.Errorf("bad osc command (%s), expected usage:\n%s", arg1, usage())
			}
			port, host, err := getPortHost(args[2])
			if err != nil {
				return nil, err
			}
			fmt.Printf("porthost = %d@%s\n", port, host)
			client := osc.NewClient(host, port)
			msg := osc.NewMessage(args[3]) // addr
			for _, val := range args[4:] { // remaining values
				s := fmt.Sprintf("%v", val)
				msg.Append(s) // always append as a string
			}
			err = client.Send(msg)
			if err != nil {
				return nil, fmt.Errorf("client.Send, err=%s", err.Error())
			}
			return nil, nil

		default:
			return nil, fmt.Errorf("bad osc command (%s), expected usage:\n%s", arg1, usage())
		}

	case "get", "getboot":
		if len(args) != 2 {
			return nil, fmt.Errorf("bad %s command, expected usage:\n%s", api, usage())
		}
		return kit.LocalEngineApi("global."+api, "name", args[1])

	case "set", "setboot":
		if len(args) != 3 {
			return nil, fmt.Errorf("bad %s command, expected usage:\n%s", api, usage())
		}
		return kit.LocalEngineApi("global."+api, "name", args[1], "value", args[2])

	case "setbootfromcurrent": // secret?
		return kit.LocalEngineApi("global." + api)

	case "env":
		path := kit.ConfigFilePath(".env")
		myenv, err := godotenv.Read(path)
		if err != nil {
			myenv = make(map[string]string)
		}
		if arg1 == "" {
			s := ""
			for k, v := range myenv {
				s = s + k + "=" + v + "\n"
			}
			return map[string]string{"result": s}, nil
		}
		switch args[1] {
		case "set":
			if len(args) < 4 {
				return nil, fmt.Errorf("not enough arguments to env command")
			}
			myenv[args[2]] = args[3]
			err = godotenv.Write(myenv, path)
			if err != nil {
				kit.LogError(err)
				return map[string]string{"error": err.Error()}, nil
			} else {
				s := args[2] + "=" + args[3] + "\n"
				return map[string]string{"result": s}, nil
			}
		case "get":
			if len(args) < 3 {
				return nil, fmt.Errorf("not enough arguments to env command")
			}
			gotten, ok := myenv[args[2]]
			if !ok {
				return map[string]string{"error": "No value"}, nil
			} else {
				return map[string]string{"result": gotten}, nil
			}
		default:
			return nil, fmt.Errorf("unknown env subcommand - %s", arg1)
		}

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
					return kit.LocalEngineApi("global.set", "name", param, "value", "true")
				}
			}
			return nil, fmt.Errorf("process %s is disabled or unknown", arg1)
		}

	case "stop":

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
			return kit.LocalEngineApi("global.set", "name", param, "value", "false")
		}

	case "version":
		s := kit.GetPaletteVersion()
		return map[string]string{"result": s}, nil

	case "mmtt":
		switch arg1 {
		case "align":
			return kit.MmttApi("align_start")
		default:
			return nil, fmt.Errorf("unknown mmtt command - %s", arg1)
		}

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
		switch arg1 {
		case "":
			return kit.LocalEngineApi("quad.test", "ntimes", "40")
		case "long":
			return kit.LocalEngineApi("quad.test", "ntimes", "400")
		case "center":
			return kit.LocalEngineApi("quad.test", "ntimes", "1000", "testtype", "center")
		default:
			return nil, fmt.Errorf("unknown test type - %s", arg1)
		}

	case "obs":
		kit.LogInfo("palette: obs command")
		err := kit.ObsCommand(arg1)
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		// return map[string]string{"result": ""}, nil
		return nil, nil

	case "hub":
		if len(args) < 2 {
			return nil, fmt.Errorf("hub command missing argument")
		}
		// No matter what, connect to the remote
		err := kit.NatsConnectRemote()
		if err != nil {
			return map[string]string{"error": err.Error()}, nil
		}
		switch arg1 {

		case "streams":
			streams, err := kit.NatsStreams()
			if err != nil {
				return map[string]string{"error": err.Error()}, nil
			}
			s := ""
			for _, stream := range streams {
				s += fmt.Sprintf("%s\n", stream)
			}
			return map[string]string{"result": s}, nil

		case "dumpraw":
			streamName := "from_palette"
			if len(args) > 2 {
				streamName = args[2]
			}
			type DumpData struct {
				Subject string `json:"subject"`
				Tm      string `json:"time"`
				Data    string `json:"data"`
			}
			err := kit.NatsDump(streamName, func(tm time.Time, subj string, data string) {
				dd := DumpData{
					Subject: subj,
					Tm:      tm.Format(kit.PaletteTimeLayout),
					Data:    data,
				}
				jsonData, err := json.Marshal(dd)
				if err != nil {
					fmt.Println("Error marshalling JSON:", err)
					return
				}

				fmt.Println(string(jsonData))
			})
			if err != nil {
				return map[string]string{"error": err.Error()}, nil
			}
			return map[string]string{"result": ""}, nil

		case "dumpload":

			streamName := "from_palette"
			err = kit.NatsDump(streamName, func(tm time.Time, subj string, data string) {

				// We only look at .load messages
				if !strings.HasSuffix(subj, ".load") {
					return
				}

				var toplevel map[string]any
				err := json.Unmarshal([]byte(data), &toplevel)
				if err != nil {
					return
				}
				host := strings.TrimPrefix(subj, streamName+".")
				host = strings.TrimSuffix(host, ".load")

				// We used to include an attractmode flag in the published .load message,
				// but now we don't; we assume that attractmode loads won't even be published.
				// This code handles old logs that have the explicit attractmode value.
				a := toplevel["attractmode"]
				if a != nil {
					attractMode, ok := a.(bool)
					if !ok {
						kit.LogError(fmt.Errorf("bad attractmode value"))
						return
					}
					// If we're in attract mode, we ignore the load
					if attractMode {
						return
					}
				}

				f := toplevel["filename"]
				filename, ok := f.(string)
				if !ok {
					kit.LogError(fmt.Errorf("bad filename value"))
					return
				}
				if filename == "_Current" {
					return
				}

				c := toplevel["category"]
				category, ok := c.(string)
				if !ok {
					kit.LogError(fmt.Errorf("bad category value"))
					return
				}

				// fmt.Printf("event=load host=%s time=%s category=%s file=%s\n",
				// 	host, tm.Format(format), category, filename)

				type DumpData struct {
					Event    string `json:"event"`
					Host     string `json:"host"`
					Category string `json:"category"`
					Tm       string `json:"time"`
					Filename string `json:"filename"`
				}

				dd := DumpData{
					Event:    "load",
					Host:     host,
					Tm:       tm.Format(kit.PaletteTimeLayout),
					Category: category,
					Filename: filename,
				}
				jsonData, err := json.Marshal(dd)
				if err != nil {
					fmt.Println("Error marshalling JSON:", err)
					return
				}

				fmt.Println(string(jsonData))

			})
			if err != nil {
				return map[string]string{"error": err.Error()}, nil
			}
			return map[string]string{"result": ""}, nil

		default:
			return nil, fmt.Errorf("nats bad subcommand")
		}

		/*
			result, err = kit.EngineNatsApi(args[1], args[2])
			if err != nil {
				return map[string]string{"error": err.Error()}, nil
			} else {
				return map[string]string{"result": result}, nil
			}
		*/

		/*
			case "remote":
				if len(args) < 3 {
					return nil, fmt.Errorf("remote command needs 2 arguments, host and api")
				}
				host := args[1]
				api := args[2]
				result, err := kit.EngineNatsApi(host, api)
				if err != nil {
					return map[string]string{"error": err.Error()}, nil
				} else {
					return map[string]string{"result": result}, nil
				}
		*/

	default:
		if len(words) < 2 {
			return nil, fmt.Errorf("unrecognized command (%s), expected usage:\n%s", api, usage())
		} else if len(words) > 2 {
			return nil, fmt.Errorf("invalid api format, expecting {plugin}.{api}\n%s", usage())
		}
		return kit.LocalEngineApi(api, args[1:]...)
	}
}

func getPortHost(porthost string) (port int, host string, err error) {
	words := strings.Split(porthost, "@")
	switch len(words) {
	case 1: // just port number, assume LocalAddress
		port, err = strconv.Atoi(words[0])
		if err == nil {
			host = kit.LocalAddress
		}
	case 2: // port@host
		host = words[1]
		port, err = strconv.Atoi(words[0])
	default:
		err = fmt.Errorf("bad format of port@host")
	}
	return port, host, err
}

func ListenAndPrint(host string, port int) {

	source := fmt.Sprintf("%s:%d", host, port)

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		fmt.Printf("received msg = %v\n", msg)
	})
	if err != nil {
		kit.LogIfError(err)
	}

	server := &osc.Server{
		Addr:       source,
		Dispatcher: d,
	}
	// ListenAndServer listens forever
	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("err = %v\n", err)
		kit.LogError(err)
		return
	}
}

func StatusOutput() (statusOut string, numRunning int) {
	s := ""
	nrunning := 0
	running, err := kit.MonitorIsRunning()
	if err == nil && running {
		s += "Monitor is running.\n"
		nrunning++
	}

	type Runnable struct {
		processName string
		userName    string
	}
	var Runnables = []Runnable{
		{"engine", "Engine"},
		{"gui", "GUI"},
		{"bidule", "Bidule"},
		{"obs", "OBS"},
		{"chat", "Chat monitor"},
		{"resolume", "Resolume"},
	}
	for _, r := range Runnables {
		running, err = kit.IsRunning(r.processName)
		if err == nil && running {
			s += (r.userName + " is running.\n")
			nrunning++
		}
	}

	/*
		b, _ := kit.GetParamBool("global.keykitrun")
		if b {
			if kit.IsRunning("keykit") {
				s += "Keykit is running.\n"
				nrunning++
			}
		}
	*/

	mmtt := os.Getenv("PALETTE_MMTT")
	if mmtt != "" && mmtt != "none" {
		running, err := kit.IsRunning("mmtt")
		if err == nil && running {
			s += "MMTT is running.\n"
			nrunning++
		}
	}

	return s, nrunning
}

func doStartEngine() error {
	running, err := kit.IsRunning("engine")
	if err == nil && running {
		return fmt.Errorf("engine is already running")
	}
	fullexe := filepath.Join(kit.PaletteDir(), "bin", kit.EngineExe)
	kit.LogInfo("palette: starting engine", "EngineExe", kit.EngineExe)
	return kit.StartExecutableLogOutput("engine", fullexe)
}

func doStartMonitor() error {
	running, err := kit.MonitorIsRunning()
	if err == nil && running {
		return fmt.Errorf("monitor is already running")
	}
	// palette_monitor.exe will restart the engine,
	// which then starts whatever global.process.* specifies.
	kit.LogInfo("palette: starting monitor", "MonitorExe", kit.MonitorExe)
	fullexe := filepath.Join(kit.PaletteDir(), "bin", kit.MonitorExe)
	return kit.StartExecutableLogOutput("monitor", fullexe)
}
