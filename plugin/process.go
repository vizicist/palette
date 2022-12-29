package plugin

import (
	"fmt"
	"strings"

	"github.com/vizicist/palette/engine"
)

type processInfo struct {
	Exe       string // just the last part
	FullPath  string
	Arg       string
	Activate  func()
	Activated bool
}

type ProcessManager struct {
	// ctx  *engine.PluginContext
	info map[string]*processInfo
}

func NewProcessInfo(exe, fullPath, arg string, activate func()) *processInfo {
	return &processInfo{
		Exe:       exe,
		FullPath:  fullPath,
		Arg:       arg,
		Activate:  activate,
		Activated: false,
	}
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		info: make(map[string]*processInfo),
	}
}

func (pm *ProcessManager) checkAutostartProcesses(ctx *engine.PluginContext) {
	autostart := engine.ConfigValueWithDefault("autostart", "")
	if autostart == "" || autostart == "nothing" || autostart == "none" {
		return
	}
	// NOTE: the autostart value can be "all"
	processes := strings.Split(autostart, ",")
	// Only check the autostart values, not all processes
	for _, autoName := range processes {
		for processName, pi := range pm.info {
			if autoName == "*" || autoName == "all" || autoName == processName {
				if !pm.isRunning(ctx, processName) {
					pm.StartRunning(ctx, processName)
					pi.Activated = false
				}
				// Even if it's already running, we
				// want to Activate it the first time we check.
				// Also, if we restart it
				if pi.Activate != nil && !pi.Activated {
					engine.LogInfo("Calling Activate", "process", processName)
					go pi.Activate()
					pi.Activated = true
				}
			}
		}
	}
}

func (pm ProcessManager) ActivateAll() {
	for _, pi := range pm.info {
		if pi.Activate != nil {
			go pi.Activate()
			pi.Activated = true
		}
	}
}

func (pm ProcessManager) AddProcess(name string, info *processInfo) {
	if info == nil {
		engine.LogWarn("Addprocess: info not available", "process", name)
	} else {
		pm.info[name] = info
	}

}

func (pm ProcessManager) StartRunning(ctx *engine.PluginContext, process string) error {

	for nm, pi := range pm.info {
		if process == "all" || nm == process {
			if pi == nil {
				return fmt.Errorf("StartRunning: no processInfo for process=%s", process)
			}
			if pi.FullPath == "" {
				return fmt.Errorf("StartRunning: unable to start %s, no executable path", process)
			}

			engine.LogInfo("StartRunning", "path", pi.FullPath)

			err := ctx.StartExecutableLogOutput(process, pi.FullPath, true, pi.Arg)
			if err != nil {
				return fmt.Errorf("start: process=%s err=%s", process, err)
			}
		}
	}
	return nil
}

func (pm ProcessManager) StopRunning(ctx *engine.PluginContext, process string) (err error) {
	for nm, pi := range pm.info {
		if process == "all" || nm == process {
			e := ctx.KillExecutable(pi.Exe)
			if e != nil {
				err = e
			}
			pi.Activated = false
		}
	}
	return err
}

func (pm ProcessManager) ProcessStatus(ctx *engine.PluginContext) string {
	s := ""
	for name := range pm.info {
		if pm.isRunning(ctx, name) {
			s += fmt.Sprintf("%s is running\n", name)
		}
	}
	return s
}

/*
func (pm ProcessManager) resolumeActivate() {
	pm.agent.ActivateResolume()
}
*/

/*
func (pm ProcessManager) engineInfoInit() *processInfo {
	exe := "palette_engine.exe"
	fullpath := filepath.Join(pm.agent.PaletteDir(), "bin", exe)
	return &processInfo{exe, fullpath, "", nil}
}
*/

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

func (pm ProcessManager) isRunning(ctx *engine.PluginContext, process string) bool {
	pi, err := pm.getProcessInfo(process)
	if err != nil {
		engine.LogWarn("IsRunning: no process named", "process", process)
		return false
	}
	b := ctx.IsRunningExecutable(pi.Exe)
	return b
}

func (pm ProcessManager) killAll(ctx *engine.PluginContext) error {
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
