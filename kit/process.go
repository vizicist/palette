package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var TheProcessManager *ProcessManager

type ProcessInfo struct {
	Exe       string // just the last part
	FullPath  string
	DirPath   string
	Arg       string
	Activate  func()
	Activated bool
}

type ProcessManager struct {
	info             map[string]*ProcessInfo
	wasStarted       map[string]*atomic.Bool
	mutex            sync.Mutex
	lastProcessCheck time.Time
	processCheckSecs float64
}

func NewProcessInfo(exe, fullPath, arg string, activate func()) *ProcessInfo {
	return &ProcessInfo{
		Exe:       exe,
		FullPath:  fullPath,
		DirPath:   "",
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

func CheckAutorestartProcesses() {
	TheProcessManager.CheckAutorestartProcesses()
}

// These should come from the process list
var MonitorExe = "palette_monitor.exe"
var EngineExe = "palette_engine.exe"
var GuiExe = "palette_gui.exe"
var ChatExe = "palette_chat.exe"
var BiduleExe = "bidule.exe"
var ResolumeExe = "avenue.exe"
var KeykitExe = "key.exe"
var MmttExe = "mmtt_kinect.exe"
var ObsExe = "obs64.exe"

func KillAllExceptMonitor() {
	LogInfo("KillAll")
	// XXX - Unfortunate that this is hard-coded.
	KillExecutable(KeykitExe)
	KillExecutable(MmttExe)
	KillExecutable(BiduleExe)
	KillExecutable(ResolumeExe)
	KillExecutable(GuiExe)
	KillExecutable(EngineExe)
	KillExecutable(ObsExe)
	KillExecutable(ChatExe)
}

func IsRunning(process string) bool {
	if process == "engine" {
		return IsRunningExecutable(EngineExe)
	}
	return TheProcessManager.IsRunning(process)
}

func MonitorIsRunning() bool {
	return IsRunningExecutable(MonitorExe)
}

// func IsEngineRunning() bool {
// 	return isRunningExecutable(EngineExe)
// 	// return isRunningExecutable(EngineExe) || isRunningExecutable(EngineExeDebug)
// }

//////////////////////////////////////////////////////////////////

func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		info:             make(map[string]*ProcessInfo),
		wasStarted:       make(map[string]*atomic.Bool),
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

	for _, process := range ProcessList() {
		runit, err := GetParamBool("global.process." + process)
		if err != nil {
			LogError(err)
			continue
		}
		isRunning := pm.IsRunning(process)
		if runit {
			if !isRunning {
				LogInfo("CheckAutorestartProcesses: Restarting", "process", process)
				err := pm.StartRunning(process)
				LogIfError(err)
			}
		} else {
			if isRunning {
				LogInfo("CheckAutorestartProcesses: Stopping", "process", process)
				err := pm.StopRunning(process)
				LogIfError(err)
			}

		}
	}
}

func ProcessList() []string {
	arr := []string{}
	arr = append(arr, "gui")
	// arr = append(arr, "monitor")
	arr = append(arr, "bidule")
	arr = append(arr, "resolume")
	arr = append(arr, "obs")
	arr = append(arr, "chat")
	/*
		keykit, err := GetParamBool("global.keykitrun")
		LogIfError(err)
		if keykit {
			arr = append(arr, "keykit")
		}
	*/
	mmtt, err := GetParam("global.mmtt")
	LogIfError(err)
	if mmtt != "" {
		arr = append(arr, "mmtt")
	}
	return arr
}

func (pm *ProcessManager) AddBuiltins() {
	for _, process := range ProcessList() {
		pm.AddProcessBuiltIn(process)
	}
}

func (pm *ProcessManager) AddProcess(process string, info *ProcessInfo) {
	if info == nil {
		LogWarn("Addprocess: info not available", "process", process)
	} else {
		pm.info[process] = info
		var b atomic.Bool
		pm.wasStarted[process] = &b
		pm.wasStarted[process].Store(false)
	}

}

func (pm *ProcessManager) AddProcessBuiltIn(process string) {

	LogOfType("process", "AddProcessBuiltIn", "process", process)
	var p *ProcessInfo
	switch process {
	case "bidule":
		p = TheBidule().ProcessInfo()
	case "resolume":
		p = TheResolume().ProcessInfo()
	case "gui":
		p = GuiProcessInfo()
	/*
		case "keykit":
			p = KeykitProcessInfo()
	*/
	case "mmtt":
		p = MmttProcessInfo()
	case "obs":
		p = ObsProcessInfo()
	case "chat":
		p = ChatProcessInfo()
	}
	if p == nil {
		LogWarn("AddProcessBuiltIn: no process info for", "process", process)
	} else {
		pm.AddProcess(process, p)
	}
}

func (pm *ProcessManager) StartRunning(process string) error {

	if pm.IsRunning(process) {
		LogInfo("StartRunning: already running", "process", process)
		return nil
	}
	pi, err := pm.GetProcessInfo(process)
	if err != nil {
		return err
	}
	if pi.FullPath == "" {
		return fmt.Errorf("StartRunning: unable to start %s, no executable path", process)
	}

	if pi.DirPath != "" {
		thisDir, err := os.Getwd()
		LogIfError(err)
		LogInfo("StartRunning: changing to", "dir", pi.DirPath)
		err = os.Chdir(pi.DirPath)
		LogIfError(err)
		defer func() {
			LogInfo("StartRunning: changing back to", "dir", thisDir)
			err = os.Chdir(thisDir)
			LogIfError(err)
		}()
	}

	LogInfo("StartRunning", "path", pi.FullPath, "arg", pi.Arg, "lenarg", len(pi.Arg))

	err = StartExecutableLogOutput(process, pi.FullPath, pi.Arg)
	if err != nil {
		return fmt.Errorf("StartRunning: process=%s err=%s", process, err)
	}
	pm.wasStarted[process].Store(true)
	if pi.Activate != nil {
		LogOfType("process", "Activate", "process", process)
		go pi.Activate()
		pi.Activated = true
	}
	return nil
}

func (pm *ProcessManager) StopRunning(process string) (err error) {

	LogOfType("process", "KillProcess", "process", process)

	pi, err := pm.GetProcessInfo(process)
	if err != nil {
		return err
	}
	KillExecutable(pi.Exe)
	pi.Activated = false
	pm.wasStarted[process].Store(false)
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

func (pm *ProcessManager) GetProcessInfo(process string) (*ProcessInfo, error) {
	p, ok := pm.info[process]
	if !ok || p == nil {
		err := fmt.Errorf("GetProcessInfo: no process info for %s", process)
		LogIfError(err)
		return nil, err
	}
	return p, nil
}

func (pm *ProcessManager) IsAvailable(process string) bool {
	p, err := pm.GetProcessInfo(process)
	return err == nil && p != nil && p.FullPath != ""
}

func (pm *ProcessManager) IsRunning(process string) bool {
	pi, err := pm.GetProcessInfo(process)
	if err != nil {
		LogIfError(err)
		return false
	}
	b := IsRunningExecutable(pi.Exe)
	return b
}

// Below here are functions that return ProcessInfo for various programs

func GuiProcessInfo() *ProcessInfo {
	gui, err := GetParam("global.gui")
	if err != nil {
		LogIfError(err)
		return nil
	}
	fullpath := filepath.Join(PaletteDir(), gui)
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("No Gui found, looking for", "path", fullpath)
		return nil
	}
	exe := filepath.Base(fullpath)

	// set PALETTE_GUI_LEVEL to convey it to the gui
	guilevel, err := GetParam("global.guidefaultlevel")
	if err != nil {
		guilevel = "0" // last resert
		LogIfError(err)
	}
	os.Setenv("PALETTE_GUI_LEVEL", guilevel)

	// set PALETTE_GUI_SIZE to convey it to the gui
	guisize, err := GetParam("global.guisize")
	if err != nil {
		LogIfError(err)
	} else {
		os.Setenv("PALETTE_GUI_SIZE", guisize)
	}

	return NewProcessInfo(exe, fullpath, "", nil)
}

func ChatProcessInfo() *ProcessInfo {
	fullpath := filepath.Join(paletteRoot, "bin", "palette_chat.exe")
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("No chat executable found, looking for", "path", fullpath)
		return nil
	}
	exe := filepath.Base(fullpath)
	return NewProcessInfo(exe, fullpath, "", nil)
}

/*
func KeykitProcessInfo() *ProcessInfo {

	// Allow parameter to override keyroot
	keyroot, err := GetParam("global.keykitroot")
	LogIfError(err)
	if keyroot == "" {
		keyroot = filepath.Join(PaletteDir(), "keykit")
	}
	LogOfType("keykit", "Setting KEYROOT", "keyroot", keyroot)
	os.Setenv("KEYROOT", keyroot)

	// Allow parameter to override keypath
	keypath, err := GetParam("global.keykitpath")
	LogIfError(err)
	if keypath == "" {
		kp1 := filepath.Join(PaletteDataPath(), "keykit", "liblocal")
		kp2 := filepath.Join(PaletteDir(), "keykit", "lib")
		keypath = kp1 + ";" + kp2
	}
	os.Setenv("KEYPATH", keypath)
	LogOfType("keykit", "Setting KEYPATH", "keypath", keypath)

	// Allow parameter to override keyoutput
	keyoutput, err := GetParam("global.keykitoutput")
	LogIfError(err)
	os.Setenv("KEYOUT", keyoutput)
	LogOfType("keykit", "Setting KEYPOUT", "keyoutput", keyoutput)

	// Allow parameter to override keyoutput
	keyallow, err := GetParam("global.keykitallow")
	LogIfError(err)
	os.Setenv("KEYALLOW", keyallow)
	LogOfType("keykit", "Setting KEYALLOW", "keyallow", keyallow)

	// Allow parameter to override path to key.exe
	fullpath, err := GetParam("global.keykitpath")
	LogIfError(err)
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("global.keykit value doesn't exist, was looking for", "fullpath", fullpath)
	}
	if fullpath == "" {
		fullpath = filepath.Join(PaletteDir(), "keykit", "bin", KeykitExe)
		if !FileExists(fullpath) {
			LogWarn("Keykit not found in default location", "fullpath", fullpath)
			return nil
		}
	}
	exe := filepath.Base(fullpath)
	return NewProcessInfo(exe, fullpath, "", nil)
}
*/

func MmttProcessInfo() *ProcessInfo {

	/*
		// The value of mmtt is either "kinect" or "oak" or ""
		mmtt, err := GetParam("global.mmtt")
		LogIfError(err)
		if mmtt == "" {
			return nil
		}
	*/
	// Should probably get this from an environment variable
	// e.g. PALETTE_MMTT
	mmtt := os.Getenv("PALETTE_MMTT")
	if mmtt == "" {
		mmtt = "kinect"
	}
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !FileExists(fullpath) {
		LogWarn("no mmtt executable found, looking for", "fullpath", fullpath)
		fullpath = ""
	}
	return NewProcessInfo("mmtt_"+mmtt+".exe", fullpath, "", nil)
}
