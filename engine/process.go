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
	// ctx  *engine.PluginContext
	info             map[string]*ProcessInfo
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
	return TheProcessManager.StartRunning(process)
}

func StopRunning(process string) error {
	return TheProcessManager.StopRunning(process)
}

func CheckAutostartProcesses() {
	TheProcessManager.CheckAutostartProcesses()
}

//////////////////////////////////////////////////////////////////

func NewProcessManager() *ProcessManager {
	processCheckSecs := TheEngine.EngineParamFloatWithDefault("engine.processchecksecs", 60)
	pm := &ProcessManager{
		info:             make(map[string]*ProcessInfo),
		lastProcessCheck: time.Time{},
		processCheckSecs: processCheckSecs,
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
			CheckAutostartProcesses()
		}()
		pm.lastProcessCheck = now
	}
}

func (pm *ProcessManager) CheckAutostartProcesses() {

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	autostart := TheEngine.Get("engine.autostart")
	if autostart == "" || autostart == "nothing" || autostart == "none" {
		return
	}
	// NOTE: the autostart value can be "all"
	processes := strings.Split(autostart, ",")
	// Only check the autostart values, not all processes
	for _, autoName := range processes {
		for processName, pi := range pm.info {
			if autoName == "*" || autoName == "all" || autoName == processName {
				if !pm.IsRunning(processName) {
					LogError(pm.StartRunning(processName))
					pi.Activated = false
				}
				// Even if it's already running, we
				// want to Activate it the first time we check.
				// Also, if we restart it
				if pi.Activate != nil && !pi.Activated {
					LogInfo("Calling Activate", "process", processName)
					go pi.Activate()
					pi.Activated = true
				}
			}
		}
	}
}

func (pm *ProcessManager) Activate(process string) error {
	doall := (process == "all" || process == "*")
	if !doall {
		_, ok := pm.info[process]
		if !ok {
			return fmt.Errorf("ProcessManager.Activate: unknown process %s", process)
		}
	}
	for nm, pi := range pm.info {
		if (doall || nm == process) && pi.Activate != nil {
			LogOfType("process", "Activate", "process", nm)
			go pi.Activate()
			pi.Activated = true
		}
	}
	return nil
}

func (pm *ProcessManager) AddProcess(name string, info *ProcessInfo) {
	if info == nil {
		LogWarn("Addprocess: info not available", "process", name)
	} else {
		pm.info[name] = info
	}

}

func AddProcessBuiltIn(process string) {
	TheProcessManager.AddProcessBuiltIn(process)
}

func (pm *ProcessManager) AddProcessBuiltIn(process string) {

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
		if mmtt := MmttInfo(); mmtt != nil {
			pm.AddProcess(process, mmtt)
		}
	}
}

func (pm *ProcessManager) StartRunning(process string) error {

	keyroot := os.Getenv("KEYROOT")
	LogOfType("process", "StartRunning", "process", process, "keyroot", keyroot)

	for nm, pi := range pm.info {
		if process == "all" || nm == process {
			if pi == nil {
				return fmt.Errorf("StartRunning: no ProcessInfo for process=%s", process)
			}
			if pi.FullPath == "" {
				return fmt.Errorf("StartRunning: unable to start %s, no executable path", process)
			}

			LogInfo("StartRunning", "path", pi.FullPath, "arg", pi.Arg, "lenarg", len(pi.Arg))

			err := StartExecutableLogOutput(process, pi.FullPath, true, pi.Arg)
			if err != nil {
				return fmt.Errorf("start: process=%s err=%s", process, err)
			}
		}
	}
	return nil
}

func (pm *ProcessManager) StopRunning(process string) (err error) {

	LogOfType("process", "StopRunning", "process", process)

	for nm, pi := range pm.info {
		if process == "all" || nm == process {
			e := killExecutable(pi.Exe)
			if e != nil {
				err = e
			}
			pi.Activated = false
		}
	}
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
	if !ok {
		return nil, fmt.Errorf("getProcessInfo: no process %s", process)
	}
	if p == nil {
		return nil, fmt.Errorf("getProcessInfo: no process info for %s", process)
	}
	return p, nil
}

func (pm *ProcessManager) IsRunning(process string) bool {
	pi, err := pm.getProcessInfo(process)
	if err != nil {
		LogWarn("IsRunning: no process named", "process", process)
		return false
	}
	b := isRunningExecutable(pi.Exe)
	return b
}

/*
func (pm ProcessManager) KillAll(ctx *PluginContext) error {
	for nm, pi := range pm.info {
		if nm != "engine" {
			err := ctx.KillExecutable(pi.Exe)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
*/

/*
// KillProcess kills a process (synchronously)

	func KillProcess(process string) {
		switch process {
		case "all":
			for _, info := range ProcessInfo {
			for _, info := range ProcssInfo {
				KillExecuable(info.Exe)
		cae "apps":
			for nm, inf := rang ProcessInfo {
				if IsAppName(nm) {
					illExecutable(ino.Exe)

			}
		deault:
			p, err = getProessInfo(process)
			if err != nil {
				illExecutablep.Exe)

}
*/

func GuiProcessInfo() *ProcessInfo {
	fullpath := TheEngine.Get("engine.gui")
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
	keyroot := TheEngine.Get("engine.keyroot")
	if keyroot == "" {
		keyroot = filepath.Join(PaletteDir(), "keykit")
	}
	os.Setenv("KEYROOT", keyroot)

	// Allow parameter to override keypath
	keypath := TheEngine.Get("engine.keypath")
	if keypath == "" {
		kp1 := filepath.Join(PaletteDataPath(), "keykit", "liblocal")
		kp2 := filepath.Join(PaletteDir(), "keykit", "lib")
		keypath = kp1 + ";" + kp2

	}
	os.Setenv("KEYPATH", keypath)

	// Allow parameter to override keyoutput
	keyoutput := TheEngine.Get("engine.keyoutput")
	if keyoutput == "" {
		keyoutput = DefaultKeykitOutput
	}
	os.Setenv("KEYOUT", keyoutput)

	// Allow parameter to override keyoutput
	keyallow := TheEngine.Get("engine.keyallow")
	if keyallow == "" {
		keyallow = "127.0.0.1"
	}
	os.Setenv("KEYALLOW", keyallow)

	// Allow parameter to override path to key.exe
	fullpath := TheEngine.Get("engine.keykit")
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("engine.keykit value doesn't exist, was looking for", "fullpath", fullpath)
	}
	if fullpath == "" {
		fullpath = DefaultKeykitPath
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
