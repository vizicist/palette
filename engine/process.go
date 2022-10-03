package engine

import (
	"fmt"
	"log"
	"os"
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
	EngineInfoInit()
	ResolumeInfoInit()
	GuiInfoInit()
	BiduleInfoInit()
	MmttInfoInit()
	AppsInfoInit()
}

func BiduleInfoInit() {
	path := ConfigValueWithDefault("bidule", "")
	if path == "" {
		return
	}
	if !FileExists(path) {
		log.Printf("No bidule found, looking for %s", path)
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
	ProcessInfo["bidule"] = &processInfo{exe, path, ConfigFilePath(bidulefile), biduleActivate}
}

func ResolumeInfoInit() {
	fullpath := ConfigValue("resolume")
	if fullpath != "" && !FileExists(fullpath) {
		log.Printf("No Resolume found, looking for %s)", fullpath)
		return
	}
	if fullpath == "" {
		fullpath = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
		if !FileExists(fullpath) {
			fullpath = "C:\\Program Files\\Resolume Arena\\Arena.exe"
			if !FileExists(fullpath) {
				log.Printf("no Resolume found in default locations")
				return
			}
		}
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	ProcessInfo["resolume"] = &processInfo{exe, fullpath, "", resolumeActivate}
}

func GuiInfoInit() {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(PaletteDir(), "bin", "pyinstalled", exe)
	ProcessInfo["gui"] = &processInfo{exe, fullpath, "", nil}
}

func EngineInfoInit() {
	exe := "palette_engine.exe"
	fullpath := filepath.Join(PaletteDir(), "bin", exe)
	ProcessInfo["engine"] = &processInfo{exe, fullpath, "", nil}
}

func MmttInfoInit() {
	// NOTE: it's inside a sub-directory of bin, so all the necessary .dll's are contained
	mmtt := ConfigStringWithDefault("mmtt", "")
	if mmtt == "" {
		return
	}
	// The value of mmtt is either "kinect" or "oak"
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !FileExists(fullpath) {
		log.Printf("no mmtt executable found, looking for %s", fullpath)
		return
	}
	ProcessInfo["mmtt debugger"] = &processInfo{"mmtt_" + mmtt + ".exe", fullpath, "", nil}
}

func IsAppName(nm string) bool {
	return strings.HasPrefix(nm, "app_")
}

func AppsInfoInit() {
	// Collect a list of all the app_*.exe files
	binpath := filepath.Join(PaletteDir(), "bin")
	findexe := func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".exe") {
			exe := filepath.Base(path)
			appname := strings.TrimSuffix(exe, ".exe")
			if IsAppName(appname) {
				ProcessInfo[appname] = &processInfo{exe, path, "", nil}
			}
		}
		return nil
	}
	err := filepath.Walk(binpath, findexe)
	if err != nil {
		log.Printf("filepath.Walk: err=%s\n", err)
	}
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

func KillAll() {
	for nm, info := range ProcessInfo {
		if nm != "engine" {
			KillExecutable(info.Exe)
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
