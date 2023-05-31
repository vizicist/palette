package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var TheProcessManager *ProcessManager

type ProcessInfo struct {
	Exe       string // just the last part
	FullPath  string
	Arg       string
	Activate  func()
	Activated bool
}

type ProcessManager struct {
	info             map[string]*ProcessInfo
	wasStarted       map[string]bool
	mutex            sync.Mutex
	lastProcessCheck time.Time
	processCheckSecs float64
}

func NewProcessInfo(exe, fullPath, arg string, activate func()) *ProcessInfo {
	return &ProcessInfo{
		Exe:       exe,
		FullPath:  fullPath,
		Arg:       arg,
		Activate:  activate,
		Activated: false,
	}
}

func StartRunning(process string) error {
	err := TheProcessManager.StartRunning(process)
	if err != nil {
		LogIfError(err)
		return (err)
	}
	return nil
}

func StopRunning(process string) error {
	return TheProcessManager.StopRunning(process)
}

func CheckAutorestartProcesses() {
	TheProcessManager.CheckAutorestartProcesses()
}

//////////////////////////////////////////////////////////////////

func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		info:             make(map[string]*ProcessInfo),
		wasStarted:       make(map[string]bool),
		lastProcessCheck: time.Time{},
		processCheckSecs: 60, // default, change it with engine.processchecksecs
	}
	return pm
}

func (pm *ProcessManager) checkProcess() {

	firstTime := (pm.lastProcessCheck == time.Time{})

	// If processCheckSecs is 0, process checking is disabled
	processCheckEnabled := pm.processCheckSecs > 0
	if !processCheckEnabled {
		if firstTime {
			LogInfo("Process Checking is disabled.")
		}
		return
	}

	now := time.Now()
	sinceLastProcessCheck := now.Sub(pm.lastProcessCheck).Seconds()
	if processCheckEnabled && (firstTime || sinceLastProcessCheck > pm.processCheckSecs) {
		// Put it in background, so calling
		// tasklist or ps doesn't disrupt realtime
		// The sleep here is because if you kill something and
		// immediately check to see if it's running, it reports that
		// it's stil running.
		go func() {
			time.Sleep(2 * time.Second)
			CheckAutorestartProcesses()
		}()
		pm.lastProcessCheck = now
	}
}

// CheckAutorestartProcesses will restart processes that were
// started but are no longer running.
func (pm *ProcessManager) CheckAutorestartProcesses() {

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	autorestart := GetParamBool("engine.autorestart")
	if !autorestart {
		return
	}
	for processName := range pm.info {
		isRunning := pm.IsRunning(processName)
		if !isRunning && pm.wasStarted[processName] {
			LogInfo("CheckAutorestartProcesses: Restarting", "process", processName)
			err := pm.StartRunning(processName)
			LogIfError(err)
		}
	}
}

func ProcessList() []string {
	arr := []string{}
	arr = append(arr,"gui")
	arr = append(arr,"bidule")
	arr = append(arr,"resolume")
	if GetParamBool("engine.keykit") {
		arr = append(arr,"keykit")
	}
	if GetParam("engine.mmtt") != "" {
		arr = append(arr,"mmtt")
	}
	return arr
}

func (pm *ProcessManager) AddBuiltins() {
	for _, process := range ProcessList() {
		pm.AddProcessBuiltIn(process)
	}
}

func (pm *ProcessManager) AddProcess(name string, info *ProcessInfo) {
	if info == nil {
		LogWarn("Addprocess: info not available", "process", name)
	} else {
		pm.info[name] = info
	}

}

func (pm *ProcessManager) AddProcessBuiltIn(process string) {

	LogOfType("process","AddProcessBuiltIn", "process", process)
	switch process {
	case "bidule":
		pm.AddProcess(process, TheBidule().ProcessInfo())
	case "resolume":
		pm.AddProcess(process, TheResolume().ProcessInfo())
	case "gui":
		pm.AddProcess(process, GuiProcessInfo())
	case "keykit":
		pm.AddProcess(process, KeykitProcessInfo())
	case "mmtt":
		pm.AddProcess(process, MmttInfo())
	}
}

func (pm *ProcessManager) StartRunning(process string) error {

	if pm.IsRunning(process) {
		LogInfo("StartRunning: already running", "process", process)
		return nil
	}
	pi, err := pm.getProcessInfo(process)
	if err != nil {
		return err
	}
	if pi.FullPath == "" {
		return fmt.Errorf("StartRunning: unable to start %s, no executable path", process)
	}

	LogInfo("StartRunning", "path", pi.FullPath, "arg", pi.Arg, "lenarg", len(pi.Arg))

	err = StartExecutableLogOutput(process, pi.FullPath, true, pi.Arg)
	if err != nil {
		return fmt.Errorf("StartRunning: process=%s err=%s", process, err)
	}
	pm.wasStarted[process] = true
	if pi.Activate != nil {
		LogOfType("process", "Activate", "process", process)
		go pi.Activate()
		pi.Activated = true
	}
	return nil
}

func (pm *ProcessManager) StopRunning(process string) (err error) {

	LogOfType("process", "StopRunning", "process", process)

	pi, err := pm.getProcessInfo(process)
	if err != nil {
		return err
	}
	_ = killExecutable(pi.Exe) // ignore errors
	pi.Activated = false
	pm.wasStarted[process] = false
	return err
}

func (pm *ProcessManager) ProcessStatus() string {
	s := ""
	for name := range pm.info {
		if pm.IsRunning(name) {
			s += fmt.Sprintf("%s is running\n", name)
		}
	}
	return s
}

func (pm *ProcessManager) getProcessInfo(process string) (*ProcessInfo, error) {
	p, ok := pm.info[process]
	if !ok || p == nil {
		err := fmt.Errorf("getProcessInfo: no process info for %s", process)
		LogIfError(err)
		return nil, err
	}
	return p, nil
}

func (pm *ProcessManager) IsAvailable(process string) bool {
	p, err := pm.getProcessInfo(process)
	return err == nil && p != nil && p.FullPath != ""
}

func (pm *ProcessManager) IsRunning(process string) bool {
	pi, err := pm.getProcessInfo(process)
	if err != nil {
		LogIfError(err)
		return false
	}
	b := isRunningExecutable(pi.Exe)
	return b
}

func GuiProcessInfo() *ProcessInfo {
	fullpath := GetParam("engine.gui")
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("No Gui found, looking for", "path", fullpath)
		return nil
	}
	if fullpath == "" {
		fullpath = DefaultGuiPath
		if !FileExists(fullpath) {
			LogWarn("Gui not found in default location")
			return nil
		}
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	return NewProcessInfo(exe, fullpath, "", nil)
}

func KeykitProcessInfo() *ProcessInfo {

	// Allow parameter to override keyroot
	keyroot := GetParam("engine.keykitroot")
	if keyroot == "" {
		keyroot = filepath.Join(PaletteDir(), "keykit")
	}
	LogInfo("Setting KEYROOT", "keyroot", keyroot)
	os.Setenv("KEYROOT", keyroot)

	// Allow parameter to override keypath
	keypath := GetParam("engine.keykitpath")
	if keypath == "" {
		kp1 := filepath.Join(PaletteDataPath(), "keykit", "liblocal")
		kp2 := filepath.Join(PaletteDir(), "keykit", "lib")
		keypath = kp1 + ";" + kp2

	}
	os.Setenv("KEYPATH", keypath)
	LogInfo("Setting KEYPATH", "keypath", keypath)

	// Allow parameter to override keyoutput
	keyoutput := GetParam("engine.keykitoutput")
	if keyoutput == "" {
		keyoutput = DefaultKeykitOutput
	}
	os.Setenv("KEYOUT", keyoutput)
	LogInfo("Setting KEYPOUT", "keyoutput", keyoutput)

	// Allow parameter to override keyoutput
	keyallow := GetParam("engine.keykitallow")
	if keyallow == "" {
		keyallow = "127.0.0.1"
	}
	os.Setenv("KEYALLOW", keyallow)
	LogInfo("Setting KEYALLOW", "keyallow", keyallow)

	// Allow parameter to override path to key.exe
	fullpath := GetParam("engine.keykitpath")
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("engine.keykit value doesn't exist, was looking for", "fullpath", fullpath)
	}
	if fullpath == "" {
		fullpath = filepath.Join(PaletteDir(), "keykit", "bin", "key.exe")
		if !FileExists(fullpath) {
			LogWarn("Keykit not found in default location", "fullpath", fullpath)
			return nil
		}
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	return NewProcessInfo(exe, fullpath, "", nil)
}

func MmttInfo() *ProcessInfo {

	// The value of mmtt is either "kinect" or "oak" or ""
	mmtt := GetParam("engine.mmtt")
	if mmtt == "" {
		return nil
	}
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !FileExists(fullpath) {
		// LogWarn("no mmtt executable found, looking for", "path", fullpath)
		fullpath = ""
	}
	return NewProcessInfo("mmtt_"+mmtt+".exe", fullpath, "", nil)
}