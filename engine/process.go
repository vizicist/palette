package engine

import (
	"fmt"
	"strings"
	"sync"
)

type ProcessInfo struct {
	Exe       string // just the last part
	FullPath  string
	Arg       string
	Activate  func()
	Activated bool
}

type ProcessManager struct {
	// ctx  *engine.PluginContext
	info  map[string]*ProcessInfo
	mutex sync.Mutex
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

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		info: make(map[string]*ProcessInfo),
	}
}

func (pm *ProcessManager) CheckAutostartProcesses() {

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	autostart := ConfigValueWithDefault("autostart", "")
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

func (pm *ProcessManager) ActivateAll() {
	for _, pi := range pm.info {
		if pi.Activate != nil {
			go pi.Activate()
			pi.Activated = true
		}
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

	switch process {
	case "bidule":
		pm.AddProcess(process, TheBidule().ProcessInfo())
	case "resolume":
		pm.AddProcess(process, TheResolume().ProcessInfo())
	}
}

func (pm *ProcessManager) StartRunning(process string) error {

	for nm, pi := range pm.info {
		if process == "all" || nm == process {
			if pi == nil {
				return fmt.Errorf("StartRunning: no ProcessInfo for process=%s", process)
			}
			if pi.FullPath == "" {
				return fmt.Errorf("StartRunning: unable to start %s, no executable path", process)
			}

			LogInfo("StartRunning", "path", pi.FullPath)

			err := StartExecutableLogOutput(process, pi.FullPath, true, pi.Arg)
			if err != nil {
				return fmt.Errorf("start: process=%s err=%s", process, err)
			}
		}
	}
	return nil
}

func (pm *ProcessManager) StopRunning(process string) (err error) {
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
