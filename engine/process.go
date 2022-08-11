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
	ProcessInfo["vsthost"] = VstHostInfo()
	ProcessInfo["mmtt"] = MmttInfo()
}

func VstHostPath() string {
	return ConfigValueWithDefault("vsthost", "")
}

func VstHostInfo() *processInfo {
	path := VstHostPath()
	if path == "" {
		return nil
	}
	if !FileExists(path) {
		log.Printf("No vsthostpath found, looking for %s", path)
		return nil
	}
	exe := path
	lastslash := strings.LastIndex(exe, "\\")
	if lastslash > 0 {
		exe = exe[lastslash+1:]
	}
	vsthostfile := ConfigValueWithDefault("vsthostfile", "")
	if vsthostfile == "" {
		vsthostfile = "default.bidule"
	}
	return &processInfo{exe, path, ConfigFilePath(vsthostfile), VstHostActivate}
}

func ResolumeInfo() *processInfo {
	fullpath := ConfigValue("resolume")
	if fullpath != "" && !FileExists(fullpath) {
		log.Printf("No Resolume found, looking for %s)", fullpath)
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
	autostart := ConfigStringWithDefault("autostart", "gui")
	if autostart == "nothing" {
		return
	}
	if autostart == "all" {
		log.Printf("autostart no longer supports all, assuming gui\n")
		autostart = "gui"
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
	addr := "127.0.0.1"
	resolumePort := 7000
	resolumeClient := osc.NewClient(addr, resolumePort)
	textLayer := oneRouter.ResolumeLayerForText()
	clipnum := 1

	// do it a few times, in case Resolume hasn't started up
	for i := 0; i < 4; i++ {
		time.Sleep(5 * time.Second)
		for _, pad := range oneRouter.regionLetters {
			layernum := oneRouter.ResolumeLayerForPad(string(pad))
			if Debug.Resolume {
				log.Printf("Activating Resolume layer %d, clip %d\n", layernum, clipnum)
			}
			connectClip(resolumeClient, layernum, clipnum)
		}
		if textLayer >= 1 {
			connectClip(resolumeClient, textLayer, clipnum)
		}
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

func VstHostActivate() {
	path := VstHostPath()
	if !strings.Contains(path, "Bidule") {
		log.Printf("VstHostActivate does nothing")
		return
	}
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
