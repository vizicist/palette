package plugin

import (
	"fmt"

	"github.com/vizicist/palette/engine"
)

type processInfo struct {
	Exe      string // just the last part
	FullPath string
	Arg      string
	Activate func()
}

type ProcessManager struct {
	ctx  *engine.PluginContext
	info map[string]*processInfo
}

func NewProcessManager(ctx *engine.PluginContext) *ProcessManager {
	return &ProcessManager{
		ctx:  ctx,
		info: make(map[string]*processInfo),
	}
}

func (pm ProcessManager) AddProcess(name string, info *processInfo) {
	pm.info[name] = info

}

func (pm ProcessManager) Activate(processName string) {
	for name, pi := range pm.info {
		if processName == "*" || name == processName {
			if pi.Activate != nil {
				pi.Activate()
			}
		}
	}
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

		pm.ctx.LogInfo("StartRunning", "path", p.FullPath)

		err = pm.ctx.StartExecutableLogOutput(process, p.FullPath, true, p.Arg)
		if err != nil {
			return fmt.Errorf("start: process=%s err=%s", process, err)
		}
	}
	return nil
}

func (pm ProcessManager) StopRunning(process string) (err error) {
	if process == "all" {
		for nm := range pm.info {
			e := pm.killProcess(nm)
			if e != nil {
				err = e
			}
		}
		return err
	} else {
		return pm.killProcess(process)
	}
}

func (pm ProcessManager) killProcess(process string) error {
	p, err := pm.getProcessInfo(process)
	if err != nil {
		return err
	}
	return pm.ctx.KillExecutable(p.Exe)
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

func (pm ProcessManager) isRunning(process string) bool {
	p, err := pm.getProcessInfo(process)
	if err != nil {
		pm.ctx.LogWarn("IsRunning: no process named", "process", process)
		return false
	}
	b := pm.ctx.IsRunningExecutable(p.Exe)
	return b
}

func (pm ProcessManager) killAll() error {
	for nm, info := range pm.info {
		if nm != "engine" {
			err := pm.ctx.KillExecutable(info.Exe)
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
			KillExecutable(info.Exe)
	cae "apps":
		for nm, inf := range ProcessInfo {
			if IsAppName(nm) {
				KillExecutable(ino.Exe)
			}
		}
	deault:
		p, err = getProcessInfo(process)
		if err != nil {
			KillExecutablep.Exe)
		}
	}
}
*/
