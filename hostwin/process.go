package hostwin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vizicist/palette/kit"
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
	wasStarted       map[string]*atomic.Bool
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

func CheckAutorestartProcesses() {
	TheProcessManager.CheckAutorestartProcesses()
}

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

	autostart, err := kit.GetParam("engine.autostart")
	LogIfError(err)
	if autostart == "" {
		return
	}
	restarts := strings.Split(autostart, ",")

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, processName := range restarts {
		isRunning := pm.IsRunning(processName)
		if !isRunning {
			LogInfo("CheckAutorestartProcesses: Restarting", "process", processName)
			err := pm.StartRunning(processName)
			LogIfError(err)
		}
	}
}

func ProcessList() []string {
	arr := []string{}
	arr = append(arr, "gui")
	// arr = append(arr, "monitor")
	arr = append(arr, "bidule")
	arr = append(arr, "resolume")
	// keykit, err := kit.GetParamBool("engine.keykitrun")
	// LogIfError(err)
	// if keykit {
	// 	arr = append(arr, "keykit")
	// }
	mmtt, err := kit.GetParam("engine.mmtt")
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

	kit.LogOfType("process", "AddProcessBuiltIn", "process", process)
	switch process {
	case "bidule":
		pm.AddProcess(process, TheWinHost.ProcessInfoBidule())
	case "resolume":
		pm.AddProcess(process, TheWinHost.ProcessInfo())
	case "gui":
		pm.AddProcess(process, GuiProcessInfo())
	// case "keykit":
	// 	pm.AddProcess(process, KeykitProcessInfo())
	case "mmtt":
		pm.AddProcess(process, MmttProcessInfo())
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

	// LogInfo("StartRunning", "path", pi.FullPath, "arg", pi.Arg, "lenarg", len(pi.Arg))

	err = StartExecutableLogOutput(process, pi.FullPath, pi.Arg)
	if err != nil {
		return fmt.Errorf("StartRunning: process=%s err=%s", process, err)
	}
	pm.wasStarted[process].Store(true)
	if pi.Activate != nil {
		kit.LogOfType("process", "Activate", "process", process)
		go pi.Activate()
		pi.Activated = true
	}
	return nil
}

func (pm *ProcessManager) KillProcess(process string) (err error) {

	kit.LogOfType("process", "KillProcess", "process", process)

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

func GuiProcessInfo() *ProcessInfo {
	fullpath, err := kit.GetParam("engine.gui")
	LogIfError(err)
	if fullpath != "" && !kit.FileExists(fullpath) {
		LogWarn("No Gui found, looking for", "path", fullpath)
		return nil
	}
	exe := filepath.Base(fullpath)

	// set PALETTE_GUI_LEVEL to convey it to the gui
	guilevel, err := kit.GetParam("engine.defaultguilevel")
	if err != nil {
		guilevel = "0" // last resert
		LogIfError(err)
	}
	os.Setenv("PALETTE_GUI_LEVEL", guilevel)

	// set PALETTE_GUI_SIZE to convey it to the gui
	guisize, err := kit.GetParam("engine.guisize")
	if err != nil {
		LogIfError(err)
	} else {
		os.Setenv("PALETTE_GUI_SIZE", guisize)
	}

	return NewProcessInfo(exe, fullpath, "", nil)
}

/*
func MonitorProcessInfo() *ProcessInfo {
	fullpath, err := GetParam("engine.monitor")
	LogIfError(err)
	if fullpath != "" && !kit.FileExists(fullpath) {
		LogWarn("No Monitor found, looking for", "path", fullpath)
		return nil
	}
	exe := filepath.Base(fullpath)
	return NewProcessInfo(exe, fullpath, "", nil)
}
*/

/*
func KeykitProcessInfo() *ProcessInfo {

	// Allow parameter to override keyroot
	keyroot, err := kit.GetParam("engine.keykitroot")
	LogIfError(err)
	if keyroot == "" {
		keyroot = filepath.Join(PaletteDir(), "keykit")
	}
	LogOfType("keykit", "Setting KEYROOT", "keyroot", keyroot)
	os.Setenv("KEYROOT", keyroot)

	// Allow parameter to override keypath
	keypath, err := kit.GetParam("engine.keykitpath")
	LogIfError(err)
	if keypath == "" {
		kp1 := filepath.Join(PaletteDataPath(), "keykit", "liblocal")
		kp2 := filepath.Join(PaletteDir(), "keykit", "lib")
		keypath = kp1 + ";" + kp2
	}
	os.Setenv("KEYPATH", keypath)
	LogOfType("keykit", "Setting KEYPATH", "keypath", keypath)

	// Allow parameter to override keyoutput
	keyoutput, err := kit.GetParam("engine.keykitoutput")
	LogIfError(err)
	os.Setenv("KEYOUT", keyoutput)
	LogOfType("keykit", "Setting KEYPOUT", "keyoutput", keyoutput)

	// Allow parameter to override keyoutput
	keyallow, err := kit.GetParam("engine.keykitallow")
	LogIfError(err)
	os.Setenv("KEYALLOW", keyallow)
	LogOfType("keykit", "Setting KEYALLOW", "keyallow", keyallow)

	// Allow parameter to override path to key.exe
	fullpath, err := kit.GetParam("engine.keykitpath")
	LogIfError(err)
	if fullpath != "" && !kit.FileExists(fullpath) {
		LogWarn("engine.keykit value doesn't exist, was looking for", "fullpath", fullpath)
	}
	if fullpath == "" {
		fullpath = filepath.Join(PaletteDir(), "keykit", "bin", KeykitExe)
		if !kit.FileExists(fullpath) {
			LogWarn("Keykit not found in default location", "fullpath", fullpath)
			return nil
		}
	}
	exe := filepath.Base(fullpath)
	return NewProcessInfo(exe, fullpath, "", nil)
}
*/

func MmttProcessInfo() *ProcessInfo {

	// The value of mmtt is either "kinect" or "oak" or ""
	mmtt, err := kit.GetParam("engine.mmtt")
	LogIfError(err)
	if mmtt == "" {
		return nil
	}
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !kit.FileExists(fullpath) {
		// LogWarn("no mmtt executable found, looking for", "path", fullpath)
		fullpath = ""
	}
	return NewProcessInfo("mmtt_"+mmtt+".exe", fullpath, "", nil)
}
