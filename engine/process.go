package engine

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type processInfo struct {
	Exe           string // just the last part
	FullPath      string
	Arg           string
	ActivateTimes int
}

var ProcessInfo = map[string](*processInfo){}

func InitProcessInfo() {
	ProcessInfo["resolume"] = ResolumeInfo()
	ProcessInfo["gui"] = GuiInfo()
	ProcessInfo["bidule"] = BiduleInfo()
}

func BiduleInfo() *processInfo {
	fullpath := ConfigValue("bidule")
	if fullpath == "" {
		fullpath = "C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_X64.exe"
	}
	if !FileExists(fullpath) {
		log.Printf("no Bidule found, looking for %s", fullpath)
		return nil
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	arg := ConfigFilePath("palette.bidule")
	return &processInfo{exe, fullpath, arg, 10}
}

func ResolumeInfo() *processInfo {
	fullpath := ConfigValue("resolume")
	if fullpath != "" && !FileExists(fullpath) {
		log.Printf("no Resolume found, looking for %s)", fullpath)
		return nil
	}
	if fullpath == "" {
		fullpath = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
		if !FileExists(fullpath) {
			fullpath = "C:\\Program Files\\Resolume Arena\\Arena.exe"
			if !FileExists(fullpath) {
				log.Printf("no Resolume found in default locations")
				return nil
			}
		}
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	return &processInfo{exe, fullpath, "", 4}
}

func GuiInfo() *processInfo {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(PaletteDir(), "bin", "pyinstalled", exe)
	return &processInfo{exe, fullpath, "", 0}
}

func StartRunning(process string) error {

	log.Printf("StartRunning: process %s\n", process)

	switch process {

	case "gui", "bidule", "resolume":
		p := ProcessInfo[process]
		_, err := StartExecutableLogOutput(process, p.FullPath, true, p.Arg)
		if err != nil {
			return fmt.Errorf("start: bidule err=%s", err)
		}
		// bidule can take forever to load, especially on non-SSD
		go activateLater(p.ActivateTimes)

	default:
		return fmt.Errorf("start: unrecognized process %s", process)
	}

	return nil
}

func StopRunning(process string) (err error) {
	if process == "all" {
		for nm := range ProcessInfo {
			e := StopRunning(nm)
			if e != nil {
				err = e
			}
		}
		return err
	} else {
		p, ok := ProcessInfo[process]
		if !ok {
			return fmt.Errorf("StopRunning: unknown process=%s", process)
		}
		return KillExecutable(p.Exe)
	}
}

func IsRunning(process string) bool {
	p, ok := ProcessInfo[process]
	if !ok {
		log.Printf("IsRunning: no process named %s\n", process)
		return false
	}
	b := IsRunningExecutable(p.Exe)
	// log.Printf("IsRunning: process=%s exe=%s b=%v\n", process, p.Exe, b)
	return b
}

func CheckProcesses() {
	for process := range ProcessInfo {
		if !IsRunning(process) {
			StartRunning(process)
		}
	}
}

func executeAPIActivate() (string, error) {

	// handle_activate sends OSC messages to start the layers in Resolume,
	// and make sure the audio is on in Bidule.
	addr := "127.0.0.1"
	resolumePort := 7000
	bidulePort := 3210

	resolumeClient := osc.NewClient(addr, resolumePort)
	start_layer(resolumeClient, 1)
	start_layer(resolumeClient, 2)
	start_layer(resolumeClient, 3)
	start_layer(resolumeClient, 4)

	biduleClient := osc.NewClient(addr, bidulePort)
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	// log.Printf("Sending %s\n", msg.String())
	err := biduleClient.Send(msg)
	if err != nil {
		return "", fmt.Errorf("handle_activate: osc to Bidule, err=%s", err)
	}
	return "0", nil
}

func start_layer(resolumeClient *osc.Client, layer int) {
	addr := fmt.Sprintf("/composition/layers/%d/clips/1/connect", layer)
	msg := osc.NewMessage(addr)
	msg.Append(int32(1))
	// log.Printf("Sending %s\n", msg.String())
	err := resolumeClient.Send(msg)
	if err != nil {
		log.Printf("start_layer: osc to Resolume err=%s\n", err)
	}
}

// KillProcess kills a process (synchronously)
func KillProcess(process string) {
	p, ok := ProcessInfo[process]
	if !ok {
		log.Printf("KillProcess: no process named %s\n", process)
		return
	}
	KillExecutable(p.Exe)
}

func activateLater(ntimes int) {
	for i := 0; i < ntimes; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		_, err := executeAPIActivate()
		if err != nil {
			log.Printf("activateLater: err=%s\n", err)
		}
	}
}
