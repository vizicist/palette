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
	Exe          string // just the last part
	FullPath     string
	Arg          string
	ActivateFunc func()
}

var ProcessInfo = map[string](*processInfo){}

func InitProcessInfo() {
	ProcessInfo["engine"] = EngineInfo()
	ProcessInfo["VSCode debugger"] = VSCodeInfo()
	ProcessInfo["resolume"] = ResolumeInfo()
	ProcessInfo["gui"] = GuiInfo()
	ProcessInfo["bidule"] = BiduleInfo()
	ProcessInfo["mmtt"] = MmttInfo()
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
	bidulefile := ConfigValue("bidulefile")
	if bidulefile == "" {
		bidulefile = "omnisphere.bidule"
	}
	arg := ConfigFilePath(bidulefile)
	return &processInfo{exe, fullpath, arg, biduleActivate}
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
	return &processInfo{exe, fullpath, "", resolumeActivate}
}

func GuiInfo() *processInfo {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(PaletteDir(), "bin", "pyinstalled", exe)
	return &processInfo{exe, fullpath, "", nil}
}

func EngineInfo() *processInfo {
	exe := "palette_engine.exe"
	fullpath := filepath.Join(PaletteDir(), "bin", exe)
	return &processInfo{exe, fullpath, "", nil}
}

func VSCodeInfo() *processInfo {
	exe := "__debug_bin.exe"
	fullpath := ""
	return &processInfo{exe, fullpath, "", nil}
}

func MmttInfo() *processInfo {
	// NOTE: it's inside a sub-directory of bin, so all the necessary .dll's are contained
	mmtt := ConfigStringWithDefault("mmtt", "")
	if mmtt == "" {
		return nil
	}
	// The value of mmtt is either "kinect" or "oak"
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !FileExists(fullpath) {
		log.Printf("no mmtt executable found, looking for %s", fullpath)
		return nil
	}
	return &processInfo{"mmtt_" + mmtt + ".exe", fullpath, "", nil}
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

	p, err := getProcessInfo(process)
	if err != nil {
		return fmt.Errorf("StartRunning: no info for process=%s", process)
	}
	if p.FullPath == "" {
		return fmt.Errorf("StartRunning: unable to start %s, no executable path", process)
	}

	log.Printf("StartRunning: path=%s\n", p.FullPath)

	err = StartExecutableLogOutput(process, p.FullPath, true, p.Arg)
	if err != nil {
		return fmt.Errorf("start: process=%s err=%s", process, err)
	}

	if p.ActivateFunc != nil {
		go p.ActivateFunc()
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
		mmtt := ConfigStringWithDefault("mmtt", "")
		if mmtt != "" {
			autostart = autostart + ",mmtt"
		}
	}
	processes := strings.Split(autostart, ",")
	for _, process := range processes {
		p, _ := getProcessInfo(process)
		if p != nil {
			if !IsRunning(process) {
				go StartRunning(process)
			}
		}
	}
}

func ProcessStatus() string {
	s := ""
	for name := range ProcessInfo {
		if IsRunning(name) {
			s += fmt.Sprintf("%s is running\n", name)
		}
	}
	return s
}

func resolumeActivate() {
	// handle_activate sends OSC messages to start the layers in Resolume,
	// and make sure the audio is on in Bidule.
	addr := "127.0.0.1"
	resolumePort := 7000
	resolumeClient := osc.NewClient(addr, resolumePort)
	for i := 0; i < 4; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		connectClip(resolumeClient, 1, 1)
		connectClip(resolumeClient, 2, 1)
		connectClip(resolumeClient, 3, 1)
		connectClip(resolumeClient, 4, 1)
	}
}

func connectClip(resolumeClient *osc.Client, layer int, clip int) {
	addr := fmt.Sprintf("/composition/layers/%d/clips/%d/connect", layer, clip)
	msg := osc.NewMessage(addr)
	// Note: sending 0 doesn't seem to disable a clip; you need to
	// bypass the layer to turn it off
	msg.Append(int32(1))
	_ = resolumeClient.Send(msg)
}

func bypassLayer(resolumeClient *osc.Client, layer int, onoff bool) {
	addr := fmt.Sprintf("/composition/layers/%d/bypassed", layer)
	msg := osc.NewMessage(addr)
	v := 0
	if onoff {
		v = 1
	}
	msg.Append(int32(v))
	_ = resolumeClient.Send(msg)
}

func biduleActivate() {
	addr := "127.0.0.1"
	bidulePort := 3210
	biduleClient := osc.NewClient(addr, bidulePort)
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	for i := 0; i < 10; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		_ = biduleClient.Send(msg)
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
