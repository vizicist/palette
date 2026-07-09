package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	json "github.com/goccy/go-json"
	"github.com/hypebeast/go-osc/osc"
	"github.com/joho/godotenv"
	"github.com/vizicist/palette/kit"
)

var natsTarget string

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	flag.StringVar(&natsTarget, "nats", "", "Use NATS to communicate with engine at specified hostname")
	flag.Parse()
	args := flag.Args()

	// If we're doing any log commands, use stdout
	doingLogs := (len(args) > 0 && (args[0] == "log" || args[0] == "logs"))
	if doingLogs {
		kit.InitLog("")
	} else {
		kit.InitLog("palette")
	}

	if !(len(args) > 0 && args[0] == "status") {
		kit.InitKit()
	}

	// Connect to NATS if requested
	if natsTarget != "" {
		err := kit.NatsConnectRemote()
		if err != nil {
			os.Stdout.WriteString("Error connecting to NATS: " + err.Error() + "\n")
			os.Exit(1)
		}
		defer kit.NatsDisconnect()
	}

	kit.LogInfo("Palette InitLog", "args", args, "natsTarget", natsTarget)

	kit.RunCLICommand(args, CliCommand)
}

func usage() string {
	return `Usage:
	palette [-nats {hostname}] start [ {processname} ]
	palette [-nats {hostname}] stop [ {processname}]
	palette [-nats {hostname}] activate [ {processname} ]
	palette status
	palette version
	palette env [ set {name} {value} | get {name} ]
	palette record [ start | stop | list | delete {name} | upload {name} ]
	palette youtube auth
	palette log [ archive | clear | types ]
	palette summarize {logfile}
	palette osc listen {port@host}
	palette osc send {port@host} {addr} ...
	palette hub [ streams | listen | ... ]
	palette [-nats {hostname}] get [ {name} ]
	palette [-nats {hostname}] patchget {ABCD...} [ {nameprefix} ]
	palette [-nats {hostname}] set {name} {value}
	palette [-nats {hostname}] patchset {ABCD...} {name} {value}
	palette [-nats {hostname}] setboot {name} {value}
	palette [-nats {hostname}] {category}.{api} [ {argname} {argvalue} ] ...

Options:
	-nats {hostname}  Use NATS to communicate with engine at specified hostname
	`
}

// EngineAPI routes API calls to NATS or HTTP based on natsTarget
func EngineAPI(api string, args ...string) (map[string]string, error) {
	if natsTarget != "" {
		return NatsEngineAPI(api, args...)
	}
	return kit.LocalEngineAPI(api, args...)
}

// NatsEngineAPI sends an API request via NATS and returns the result
func NatsEngineAPI(api string, args ...string) (map[string]string, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("NatsEngineAPI: odd number of args, should be even")
	}

	// Build JSON request
	request := map[string]string{"api": api}
	for n := 0; n < len(args); n += 2 {
		request[args[n]] = args[n+1]
	}
	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("NatsEngineAPI: failed to marshal request: %w", err)
	}

	result, err := kit.EngineNatsAPI(natsTarget, string(jsonBytes), 10*time.Second)
	if err != nil {
		return nil, err
	}

	// Parse the JSON response
	output, err := kit.StringMap(result)
	if err != nil {
		return nil, fmt.Errorf("NatsEngineAPI: unable to parse response: %s", err)
	}
	if errstr, hasError := output["error"]; hasError {
		return nil, fmt.Errorf("%s", errstr)
	}
	return output, nil
}

// cliHandler is one top-level CLI command. args includes the command name
// itself (args[0]). Handlers return (nil, err) on failure — never a
// map with an "error" key — so main() can exit non-zero consistently.
type cliHandler func(args []string) (map[string]string, error)

// cliHandlers dispatches the first CLI argument to its handler.
// Anything not in this table falls through to the {plugin}.{api} engine form.
// Populated in init() rather than a var initializer because cmdRestart
// re-enters CliCommand, which would otherwise be an initialization cycle.
var cliHandlers map[string]cliHandler

func init() {
	cliHandlers = map[string]cliHandler{
		"status":             cmdStatus,
		"restart":            cmdRestart,
		"osc":                cmdOsc,
		"get":                cmdGetBoot,
		"getboot":            cmdGetBoot,
		"patchget":           cmdPatchGet,
		"patchset":           cmdPatchSet,
		"record":             cmdRecord,
		"set":                cmdSetBoot,
		"setboot":            cmdSetBoot,
		"setbootfromcurrent": cmdSetBootFromCurrent, // secret?
		"env":                cmdEnv,
		"start":              cmdStart,
		"stop":               cmdStop,
		"activate":           cmdActivate,
		"version":            cmdVersion,
		"mmtt":               cmdMmtt,
		"log":                cmdLog,
		"logs":               cmdLog,
		"test":               cmdTest,
		"obs":                cmdObs,
		"youtube":            cmdYouTube,
		"hub":                cmdHub,
	}
}

func CliCommand(args []string) (map[string]string, error) {

	if len(args) == 0 {
		return nil, fmt.Errorf("%s", usage())
	}

	api := args[0]
	if handler, ok := cliHandlers[api]; ok {
		return handler(args)
	}

	// Not a built-in command: treat it as a {plugin}.{api} engine call.
	words := strings.Split(api, ".")
	if len(words) < 2 {
		return nil, fmt.Errorf("unrecognized command (%s), expected usage:\n%s", api, usage())
	} else if len(words) > 2 {
		return nil, fmt.Errorf("invalid api format, expecting {plugin}.{api}\n%s", usage())
	}
	return EngineAPI(api, args[1:]...)
}

// cliArg1 returns args[1] or "" — the optional subcommand argument.
func cliArg1(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	return ""
}

func cmdStatus(args []string) (map[string]string, error) {
	s, nrunning := StatusOutput()
	if nrunning == 0 {
		s = "No palette processes are running."
	}
	return map[string]string{"result": s}, nil
}

func cmdRestart(args []string) (map[string]string, error) {
	_, err := CliCommand([]string{"stop"})
	if err != nil {
		return nil, err
	}
	_, err = CliCommand([]string{"start"})
	if err != nil {
		return nil, err
	}
	return map[string]string{"result": "RESTARTED"}, nil
}

func cmdOsc(args []string) (map[string]string, error) {
	arg1 := cliArg1(args)
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
}

func cmdGetBoot(args []string) (map[string]string, error) {
	if len(args) > 2 {
		return nil, fmt.Errorf("bad %s command, expected usage:\n%s", args[0], usage())
	}
	name := ""
	if len(args) == 2 {
		name = args[1]
	}
	// Use the ...withprefix apis to get all matching parameters
	return EngineAPI("global."+args[0]+"withprefix", "name", name)
}

func cmdPatchGet(args []string) (map[string]string, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("bad patchget command, expected usage:\n%s", usage())
	}
	category := ""
	if len(args) == 3 {
		category = args[2]
	}
	if args[1] == "*" {
		result := strings.Builder{}
		for _, patchName := range []string{"A", "B", "C", "D"} {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			patchResult, err := patchGetOutput(patchName, category)
			if err != nil {
				return nil, err
			}
			result.WriteString(patchResult)
		}
		return map[string]string{"result": strings.TrimSuffix(result.String(), "\n")}, nil
	}
	result, err := patchGetOutput(args[1], category)
	if err != nil {
		return nil, err
	}
	return map[string]string{"result": result}, nil
}

func cmdPatchSet(args []string) (map[string]string, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("bad patchset command, expected usage:\n%s", usage())
	}
	if args[1] == "*" {
		for _, patchName := range []string{"A", "B", "C", "D"} {
			_, err := EngineAPI("patch.set", "patch", patchName, "name", args[2], "value", args[3])
			if err != nil {
				return nil, err
			}
		}
		return map[string]string{"result": ""}, nil
	}
	return EngineAPI("patch.set", "patch", args[1], "name", args[2], "value", args[3])
}

func cmdRecord(args []string) (map[string]string, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("bad record command, expected start|stop|list\n%s", usage())
	}
	switch args[1] {
	case "start":
		res, err := EngineAPI("global.obsrecord")
		if err != nil {
			return nil, err
		}
		return map[string]string{"result": formatRecordStatus(res["result"], "started")}, nil
	case "stop":
		res, err := EngineAPI("global.obsrecordstop")
		if err != nil {
			return nil, err
		}
		return map[string]string{"result": formatRecordStatus(res["result"], "stopped")}, nil
	case "list":
		res, err := EngineAPI("global.obsrecordlist")
		if err != nil {
			return nil, err
		}
		return map[string]string{"result": formatRecordList(res["result"])}, nil
	case "delete":
		if len(args) < 3 {
			return nil, fmt.Errorf("record delete needs a recording name")
		}
		_, err := EngineAPI("global.obsrecorddelete", "name", args[2])
		if err != nil {
			return nil, err
		}
		return map[string]string{"result": "Deleted " + args[2] + "\n"}, nil
	case "upload":
		if len(args) < 3 {
			return nil, fmt.Errorf("record upload needs a recording name")
		}
		_, err := EngineAPI("global.youtubeupload", "name", args[2])
		if err != nil {
			return nil, err
		}
		return map[string]string{"result": "Upload of " + args[2] + " started, watch engine.log for the result\n"}, nil
	default:
		return nil, fmt.Errorf("bad record command (%s), expected start|stop|list|delete|upload\n%s", args[1], usage())
	}
}

// cmdYouTube handles one-time YouTube authorization.  It runs in the CLI
// process (not the engine) so it can print the code and block while the
// user visits the URL; the resulting refresh token goes into the shared
// env file, where the engine reads it.
func cmdYouTube(args []string) (map[string]string, error) {
	if len(args) < 2 || args[1] != "auth" {
		return nil, fmt.Errorf("bad youtube command, expected: palette youtube auth")
	}
	err := kit.YouTubeDeviceAuth(func(verificationURL, userCode string) {
		fmt.Printf("On any device, visit:\n\n    %s\n\nand enter the code:  %s\n\nWaiting for authorization...\n", verificationURL, userCode)
	})
	if err != nil {
		return nil, err
	}
	return map[string]string{"result": "YouTube authorization complete, uploads are now enabled.\n"}, nil
}

func cmdSetBoot(args []string) (map[string]string, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("bad %s command, expected usage:\n%s", args[0], usage())
	}
	return EngineAPI("global."+args[0], "name", args[1], "value", args[2])
}

func cmdSetBootFromCurrent(args []string) (map[string]string, error) {
	return EngineAPI("global." + args[0])
}

func cmdEnv(args []string) (map[string]string, error) {
	arg1 := cliArg1(args)
	path := kit.EnvFilePath()
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
	switch arg1 {
	case "set":
		if len(args) < 4 {
			return nil, fmt.Errorf("not enough arguments to env command")
		}
		myenv[args[2]] = args[3]
		err = godotenv.Write(myenv, path)
		if err != nil {
			return nil, err
		}
		return map[string]string{"result": args[2] + "=" + args[3] + "\n"}, nil
	case "get":
		if len(args) < 3 {
			return nil, fmt.Errorf("not enough arguments to env command")
		}
		// Show the effective value: the env file (.palette/.env) if set,
		// otherwise the OS environment variable of the same name.
		gotten := kit.EnvLookup(args[2])
		if gotten == "" {
			return nil, fmt.Errorf("no value for %s", args[2])
		}
		return map[string]string{"result": gotten}, nil
	default:
		return nil, fmt.Errorf("unknown env subcommand - %s", arg1)
	}
}

func cmdStart(args []string) (map[string]string, error) {
	arg1 := cliArg1(args)
	switch arg1 {

	case "", "monitor":
		return nil, doStartMonitor()

	case "engine":
		return nil, doStartEngine("")

	case "engineonly":
		return nil, doStartEngine("engineonly")

	default:
		// Only the monitor and engine are started directly by palette.
		// The monitor will restart the engine if it dies, and
		// the engine will restart any processes specified in global.process.*.
		for _, process := range kit.ProcessList() {
			if arg1 == process {
				param := "global.process." + arg1
				return EngineAPI("global.set", "name", param, "value", "true")
			}
		}
		return nil, fmt.Errorf("process %s is disabled or unknown", arg1)
	}
}

func cmdStop(args []string) (map[string]string, error) {
	arg1 := cliArg1(args)
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
		return EngineAPI("global.set", "name", param, "value", "false")
	}
}

func cmdActivate(args []string) (map[string]string, error) {
	if len(args) > 2 {
		return nil, fmt.Errorf("bad activate command, expected usage:\n%s", usage())
	}
	process := cliArg1(args)
	if process == "" {
		process = "resolume"
	}
	return EngineAPI("global.activate", "process", process)
}

func cmdVersion(args []string) (map[string]string, error) {
	return map[string]string{"result": kit.GetPaletteVersion()}, nil
}

func cmdMmtt(args []string) (map[string]string, error) {
	arg1 := cliArg1(args)
	switch arg1 {
	case "align":
		return kit.MmttAPI("align_start")
	default:
		return nil, fmt.Errorf("unknown mmtt command - %s", arg1)
	}
}

func cmdLog(args []string) (map[string]string, error) {
	arg1 := cliArg1(args)
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
	case "types":
		return map[string]string{"result": strings.Join(kit.LogTypeNames(), "\n")}, nil
	}
	return nil, fmt.Errorf("invalid log command: %s", arg1)
}

func cmdTest(args []string) (map[string]string, error) {
	arg1 := cliArg1(args)
	switch arg1 {
	case "":
		return EngineAPI("quad.test", "ntimes", "40")
	case "long":
		return EngineAPI("quad.test", "ntimes", "400")
	case "center":
		return EngineAPI("quad.test", "ntimes", "1000", "testtype", "center")
	default:
		return nil, fmt.Errorf("unknown test type - %s", arg1)
	}
}

func cmdObs(args []string) (map[string]string, error) {
	kit.LogInfo("palette: obs command")
	if err := kit.ObsCommand(cliArg1(args)); err != nil {
		return nil, err
	}
	return nil, nil
}

func cmdHub(args []string) (map[string]string, error) {
	// Delegate to palette_hub executable with remaining arguments
	hubArgs := args[1:] // everything after "hub"
	cmd := exec.Command("palette_hub", hubArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return map[string]string{"result": ""}, nil
}

// formatRecordStatus turns the {"recording":..,"remaining":..} JSON returned by
// the obsrecord/obsrecordstop APIs into a human-readable line.
func formatRecordStatus(jsonStr, action string) string {
	var st kit.OBSRecordState
	if err := json.Unmarshal([]byte(jsonStr), &st); err != nil {
		return jsonStr // fall back to raw
	}
	switch action {
	case "started":
		if st.Recording {
			return fmt.Sprintf("Recording started (%.0f seconds).", st.Remaining)
		}
		return "Recording did not start."
	case "stopped":
		if !st.Recording {
			return "Recording stopped."
		}
		return fmt.Sprintf("Still recording (%.0f seconds remaining).", st.Remaining)
	}
	return jsonStr
}

// formatRecordList turns the JSON array returned by obsrecordlist into a
// human-readable table (one recording per line).
func formatRecordList(jsonStr string) string {
	if strings.TrimSpace(jsonStr) == "" {
		return "No recordings."
	}
	var recs []kit.OBSRecordingFile
	if err := json.Unmarshal([]byte(jsonStr), &recs); err != nil {
		return jsonStr // fall back to raw
	}
	if len(recs) == 0 {
		return "No recordings."
	}
	b := strings.Builder{}
	for i, r := range recs {
		if i > 0 {
			b.WriteString("\n")
		}
		dur := "-"
		if r.Duration > 0 {
			total := int(r.Duration + 0.5)
			dur = fmt.Sprintf("%d:%02d", total/60, total%60)
		}
		fmt.Fprintf(&b, "%-30s  %6s  %10s  %s", r.Name, dur, formatByteSize(r.Size), r.ModTime)
	}
	return b.String()
}

func formatByteSize(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", n, units[i])
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

func patchGetOutput(patchName string, category string) (string, error) {
	out, err := EngineAPI("patch.getparams", "patch", patchName, "category", category)
	if err != nil {
		return "", err
	}
	lines := strings.Split(out["result"], "\n")
	result := strings.Builder{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		result.WriteString(patchName)
		result.WriteString(".")
		result.WriteString(line)
		result.WriteString("\n")
	}
	return strings.TrimSuffix(result.String(), "\n"), nil
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
	running, err := kit.IsRunningExecutable(kit.MonitorExe)
	if err == nil && running {
		s += "Monitor is running.\n"
		nrunning++
	}

	type Runnable struct {
		exe      string
		userName string
	}
	var Runnables = []Runnable{
		{kit.EngineExe, "Engine"},
		{kit.GuiExe, "GUI"},
		{kit.BiduleExe, "Bidule"},
		{kit.ObsExe, "OBS"},
		{kit.ChatExe, "Chat monitor"},
		{kit.ResolumeExe, "Resolume"},
	}
	for _, r := range Runnables {
		if r.userName == "GUI" {
			running, err = kit.IsMacPaletteChromeRunning()
		} else {
			running, err = kit.IsRunningExecutable(r.exe)
		}
		if err == nil && running {
			s += (r.userName + " is running.\n")
			nrunning++
		}
	}
	running, err = kit.IsSamplesplitterRunning()
	if err == nil && running {
		s += "SampleSplitter is running.\n"
		nrunning++
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
		running, err := kit.IsRunningExecutable(kit.MmttExe)
		if err == nil && running {
			s += "MMTT is running.\n"
			nrunning++
		}
	}

	return s, nrunning
}

func doStartEngine(arg string) error {
	running, err := kit.IsRunning("engine")
	if err == nil && running {
		return fmt.Errorf("engine is already running")
	}
	fullexe := kit.PaletteBinaryPath(kit.EngineExe)
	kit.LogInfo("palette: starting engine", "EngineExe", kit.EngineExe, "arg", arg)
	_, err = kit.StartExecutableLogOutput("engine", fullexe, arg)
	return err
}

func doStartMonitor() error {
	running, err := kit.MonitorIsRunning()
	if err == nil && running {
		return fmt.Errorf("monitor is already running")
	}
	// palette_monitor.exe will restart the engine,
	// which then starts whatever global.process.* specifies.
	kit.LogInfo("palette: starting monitor", "MonitorExe", kit.MonitorExe)
	fullexe := kit.PaletteBinaryPath(kit.MonitorExe)
	_, err = kit.StartExecutableLogOutput("monitor", fullexe)
	return err
}
