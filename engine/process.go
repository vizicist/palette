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
	Exe               string // just the last part
	FullPath          string
	Arg               string
	ActivateLaterTime int
}

var ProcessInfo = map[string](*processInfo){}

func InitProcessInfo() {
	ProcessInfo["resolume"] = ResolumeInfo()
	ProcessInfo["gui"] = GuiInfo()
	ProcessInfo["bidule"] = BiduleInfo()
	ProcessInfo["mmtt_kinect"] = MmttInfo("mmtt_kinect")
	ProcessInfo["mmtt_kinect"] = MmttInfo("mmtt_kinect")
	ProcessInfo["mmtt_oak"] = MmttInfo("mmtt_oak")
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

func MmttInfo(mmtt string) *processInfo {
	exe := mmtt + ".exe"
	fullpath := filepath.Join(PaletteDir(), "bin", mmtt, exe)
	if !FileExists(fullpath) {
		log.Printf("no mmtt executable found, looking for %s", fullpath)
		return nil
	}
	return &processInfo{exe, fullpath, "", 0}
}

func getProcessInfo(process string) (*processInfo, error) {
	p, ok := ProcessInfo[process]
	if !ok {
		return nil, fmt.Errorf("getProcessInfo: no process %s", process)
	}
	if p == nil {
		return nil, fmt.Errorf("getProcessInfo: no process info for %s", process)
	}
	return p, nil
}

func StartRunning(process string) error {

	log.Printf("StartRunning: process %s\n", process)

	p, err := getProcessInfo(process)
	if err != nil {
		return err
	}

	log.Printf("StartExecutableLogOutput: process=%s fullpath=%s\n", process, p.FullPath)
	err = StartExecutableLogOutput(process, p.FullPath, true, p.Arg)
	if err != nil {
		return fmt.Errorf("start: process=%s err=%s", process, err)
	}

	// Some things (bidule and resolume) need to be activated
	if p.ActivateLaterTime > 0 {
		go activateLater(p.ActivateLaterTime)
	}
	return nil
}

// StopRunning doesn't return any errors
func StopRunning(process string) (err error) {
	if process == "all" {
		for nm := range ProcessInfo {
			StopRunning(nm)
		}
		return err
	} else {
		p, err := getProcessInfo(process)
		if err != nil {
			return err
		}
		KillExecutable(p.Exe)
		return nil
	}
}

func IsRunning(process string) bool {
	p, err := getProcessInfo(process)
	if err != nil {
		log.Printf("IsRunning: no process named %s\n", process)
		return false
	}
	b := IsRunningExecutable(p.Exe)
	// log.Printf("IsRunning: process=%s exe=%s b=%v\n", process, p.Exe, b)
	return b
}

func CheckProcessesAndRestartIfNecessary() {
	autostart := ConfigStringWithDefault("autostart", "all")
	if autostart == "nothing" {
		return
	}
	if autostart == "all" {
		autostart = "resolume,bidule,gui"
	}
	processes := strings.Split(autostart, ",")
	for _, process := range processes {
		p, _ := getProcessInfo(process)
		if p != nil {
			if !IsRunning(process) {
				StartRunning(process)
			}
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
	p, err := getProcessInfo(process)
	if err != nil {
		log.Printf("KillProcess: err=%s\n", err)
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
