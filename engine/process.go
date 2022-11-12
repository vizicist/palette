package engine

import (
	"fmt"
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

type ProcessManager struct {
	info map[string]*processInfo
}

func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		info: make(map[string]*processInfo),
	}
	pm.engineInfoInit()
	pm.resolumeInfoInit()
	pm.guiInfoInit()
	pm.biduleInfoInit()
	pm.mmttInfoInit()
	return pm
}

func (pm ProcessManager) StartRunning(process string) error {

	switch process {
	case "all":
		for nm := range pm.info {
			pm.StartRunning(nm)
		}
	default:
		p, err := pm.getProcessInfo(process)
		if err != nil {
			return fmt.Errorf("StartRunning: no info for process=%s", process)
		}
		if p.FullPath == "" {
			return fmt.Errorf("StartRunning: unable to start %s, no executable path", process)
		}

		Info("StartRunning", "path", p.FullPath)

		err = StartExecutableLogOutput(process, p.FullPath, true, p.Arg)
		if err != nil {
			return fmt.Errorf("start: process=%s err=%s", process, err)
		}

		if p.ActivateFunc != nil {
			go p.ActivateFunc()
		}
	}
	return nil
}

// StopRunning doesn't return any errors
func (pm ProcessManager) StopRunning(process string) (err error) {
	switch process {
	case "all":
		for nm := range pm.info {
			pm.StopRunning(nm)
		}
		TheEngine().StopMe()
		return err
	default:
		p, err := pm.getProcessInfo(process)
		if err != nil {
			return err
		}
		KillExecutable(p.Exe)
		return nil
	}
}

func (pm ProcessManager) ProcessStatus() string {
	s := ""
	for name := range pm.info {
		if pm.isRunning(name) {
			s += fmt.Sprintf("%s is running\n", name)
		}
	}
	return s
}

func (pm ProcessManager) biduleInfoInit() {
	path := ConfigValueWithDefault("bidule", "")
	if path == "" {
		return
	}
	if !FileExists(path) {
		Warn("No bidule found, looking for", "path", path)
		return
	}
	exe := path
	lastslash := strings.LastIndex(exe, "\\")
	if lastslash > 0 {
		exe = exe[lastslash+1:]
	}
	bidulefile := ConfigValueWithDefault("bidulefile", "")
	if bidulefile == "" {
		bidulefile = "default.bidule"
	}
	pm.info["bidule"] = &processInfo{exe, path, ConfigFilePath(bidulefile), biduleActivate}
}

func (pm ProcessManager) resolumeInfoInit() {
	fullpath := ConfigValue("resolume")
	if fullpath != "" && !FileExists(fullpath) {
		Warn("No Resolume found, looking for", "path", fullpath)
		return
	}
	if fullpath == "" {
		fullpath = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
		if !FileExists(fullpath) {
			fullpath = "C:\\Program Files\\Resolume Arena\\Arena.exe"
			if !FileExists(fullpath) {
				Warn("Resolume not found in default locations")
				return
			}
		}
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	pm.info["resolume"] = &processInfo{exe, fullpath, "", resolumeActivate}
}

func (pm ProcessManager) guiInfoInit() {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(PaletteDir(), "bin", "pyinstalled", exe)
	pm.info["gui"] = &processInfo{exe, fullpath, "", nil}
}

func (pm ProcessManager) engineInfoInit() {
	exe := "palette_engine.exe"
	fullpath := filepath.Join(PaletteDir(), "bin", exe)
	pm.info["engine"] = &processInfo{exe, fullpath, "", nil}
}

func (pm ProcessManager) mmttInfoInit() {
	// NOTE: it's inside a sub-directory of bin, so all the necessary .dll's are contained
	mmtt := ConfigStringWithDefault("mmtt", "")
	if mmtt == "" {
		return
	}
	// The value of mmtt is either "kinect" or "oak"
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !FileExists(fullpath) {
		Warn("no mmtt executable found, looking for", "path", fullpath)
		return
	}
	pm.info["mmtt debugger"] = &processInfo{"mmtt_" + mmtt + ".exe", fullpath, "", nil}
}

func (pm ProcessManager) getProcessInfo(process string) (*processInfo, error) {
	p, ok := pm.info[process]
	if !ok {
		return nil, fmt.Errorf("getProcessInfo: no process %s", process)
	}
	if p == nil {
		return nil, fmt.Errorf("getProcessInfo: no process info for %s", process)
	}
	return p, nil
}

func (pm ProcessManager) isRunning(process string) bool {
	p, err := pm.getProcessInfo(process)
	if err != nil {
		Warn("IsRunning: no process named", "process", process)
		return false
	}
	b := IsRunningExecutable(p.Exe)
	return b
}

func resolumeActivate() {
	// handle_activate sends OSC messages to start the layers in Resolume,
	addr := "127.0.0.1"
	resolumePort := 7000
	resolumeClient := osc.NewClient(addr, resolumePort)
	textLayer := TheEngine().Router.ResolumeLayerForText()
	clipnum := 1

	// do it a few times, in case Resolume hasn't started up
	for i := 0; i < 4; i++ {
		time.Sleep(5 * time.Second)
		for _, pad := range TheEngine().Router.playerLetters {
			layernum := TheEngine().Router.ResolumeLayerForPad(string(pad))
			DebugLogOfType("resolume", "Activating Resolume", "layer", layernum, "clipnum", clipnum)
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

func (pm ProcessManager) killAll() {
	for nm, info := range pm.info {
		if nm != "engine" {
			KillExecutable(info.Exe)
		}
	}
}

func (pm ProcessManager) checkProcessesAndRestartIfNecessary() {
	autostart := ConfigStringWithDefault("autostart", "")
	if autostart == "" || autostart == "nothing" || autostart == "none" {
		return
	}
	processes := strings.Split(autostart, ",")
	for _, process := range processes {
		p, _ := pm.getProcessInfo(process)
		if p != nil {
			if !pm.isRunning(process) {
				go pm.StartRunning(process)
			}
		}
	}
}

/*
// KillProcess kills a process (synchronously)
func KillProcess(process string) {
	switch process {
	case "all":
		for _, info := range ProcessInfo {
			KillExecutable(info.Exe)
		}
	case "apps":
		for nm, info := range ProcessInfo {
			if IsAppName(nm) {
				KillExecutable(info.Exe)
			}
		}
	default:
		p, err := getProcessInfo(process)
		if err != nil {
			KillExecutable(p.Exe)
		}
	}
}
*/
