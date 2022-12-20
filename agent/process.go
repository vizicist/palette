package agent

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
	agent *engine.AgentContext
	info  map[string]*processInfo
}

func NewProcessManager(ctx *engine.AgentContext) *ProcessManager {
	return &ProcessManager{
		agent: ctx,
		info:  make(map[string]*processInfo),
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

		pm.agent.LogInfo("StartRunning", "path", p.FullPath)

		err = pm.agent.StartExecutableLogOutput(process, p.FullPath, true, p.Arg)
		if err != nil {
			return fmt.Errorf("start: process=%s err=%s", process, err)
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
		// TheEngine().StopMe()
		return err
	default:
		p, err := pm.getProcessInfo(process)
		if err != nil {
			return err
		}
		pm.agent.KillExecutable(p.Exe)
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
		pm.agent.LogWarn("IsRunning: no process named", "process", process)
		return false
	}
	b := pm.agent.IsRunningExecutable(p.Exe)
	return b
}

func (pm ProcessManager) killAll() {
	for nm, info := range pm.info {
		if nm != "engine" {
			pm.agent.KillExecutable(info.Exe)
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
