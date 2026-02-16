package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	runningPids      map[string]int // track PIDs of running processes
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

func EmptyProcessInfo() *ProcessInfo {
	return NewProcessInfo("", "", "", nil)
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
var GuiExe = "chrome.exe"
var ChatExe = "palette_chat.exe"
var BiduleExe = "bidule.exe"
var ResolumeExe = "avenue.exe"

var MmttExe = "mmtt_kinect.exe"
var ObsExe = "obs64.exe"

func KillAllExceptMonitor() {
	LogInfo("KillAll")
	// XXX - Unfortunate that this is hard-coded.
	// KillExecutable(KeykitExe)
	KillExecutable(MmttExe)
	KillExecutable(BiduleExe)
	KillExecutable(ResolumeExe)
	KillExecutable(GuiExe)
	KillExecutable(EngineExe)
	KillExecutable(ObsExe)
	KillExecutable(ChatExe)
}

func IsRunning(process string) (bool, error) {
	if process == "engine" {
		return IsRunningExecutable(EngineExe)
	}
	return TheProcessManager.IsRunning(process)
}

func MonitorIsRunning() (bool, error) {
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
		runningPids:      make(map[string]int),
		lastProcessCheck: time.Time{},
		processCheckSecs: 60, // default
	}
	return pm
}

func (pm *ProcessManager) checkProcess() {

	firstTime := (pm.lastProcessCheck.Equal(time.Time{}))

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

	if NoProcess {
		return
	}
	for _, process := range ProcessList() {
		if !pm.IsAvailable(process) {
			continue
		}
		runit, err := GetParamBool("global.process." + process)
		if err != nil {
			LogError(err)
			continue
		}
		running, err := pm.IsRunning(process)
		if err == nil {
			if runit {
				if !running {
					LogInfo("CheckAutorestartProcesses: Restarting", "process", process)
					err := pm.StartRunning(process)
					LogIfError(err)
				}
			} else {
				// If we're not supposed to be running it, stop it
				if running {
					LogInfo("CheckAutorestartProcesses: Stopping", "process", process)
					err := pm.StopRunning(process)
					LogIfError(err)
				}
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
	arr = append(arr, "mmtt")
	/*
		keykit, err := GetParamBool("global.keykitrun")
		LogIfError(err)
		if keykit {
			arr = append(arr, "keykit")
		}
	*/
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

	// For GUI (Chrome), always kill any existing instances first
	// This ensures we start fresh and avoids issues with multiple Chrome instances
	if process == "gui" {
		LogInfo("StartRunning: killing any existing Chrome instances before starting GUI")
		KillExecutable("chrome.exe")
	}

	running, err := pm.IsRunning(process)
	if err != nil {
		return err
	}
	if running {
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

	// Log the command being executed
	LogInfo("StartRunning", "process", process, "fullPath", pi.FullPath, "arg", pi.Arg)

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

	// LogInfo("StartRunning", "path", pi.FullPath, "arg", pi.Arg, "lenarg", len(pi.Arg))

	pid, err := StartExecutableLogOutput(process, pi.FullPath, pi.Arg)
	if err != nil {
		return fmt.Errorf("StartRunning: process=%s err=%w", process, err)
	}
	pm.runningPids[process] = pid
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

	// Use PID-based killing if we have the PID (avoids killing other instances)
	if pid, ok := pm.runningPids[process]; ok && pid > 0 {
		KillByPid(pid)
		delete(pm.runningPids, process)
	} else {
		// Fall back to killing by executable name
		KillExecutable(pi.Exe)
	}

	pi.Activated = false
	pm.wasStarted[process].Store(false)
	return err
}

/*
func (pm *ProcessManager) ProcessStatus() string {
	s := ""
	for name := range pm.info {
		running, err := pm.IsRunning(name)
		if err == nil && running {
			s += fmt.Sprintf("%s is running\n", name)
		}
	}
	return s
}
*/

func (pm *ProcessManager) GetProcessInfo(process string) (*ProcessInfo, error) {
	p, ok := pm.info[process]
	if !ok || p == nil {
		err := fmt.Errorf("GetProcessInfo: no process info for %s", process)
		LogError(err)
		return nil, err
	}
	if p.Exe == "" {
		err := fmt.Errorf("GetProcessInfo: no executable info for %s", process)
		LogError(err)
	}
	if p.FullPath == "" {
		err := fmt.Errorf("GetProcessInfo: no fullpath info for %s", process)
		LogError(err)
	}
	return p, nil
}

func (pm *ProcessManager) IsAvailable(process string) bool {
	p, ok := pm.info[process]
	return ok && p != nil && p.Exe != "" && p.FullPath != ""
}

func (pm *ProcessManager) IsRunning(process string) (bool, error) {
	pi, err := pm.GetProcessInfo(process)
	if err != nil {
		return false, err
	}
	return IsRunningExecutable(pi.Exe)
}

// Below here are functions that return ProcessInfo for various programs

func GuiProcessInfo() *ProcessInfo {
	// Use Chrome to display the webui
	chromePath := `C:\Program Files\Google\Chrome\Application\chrome.exe`
	if !FileExists(chromePath) {
		LogWarn("Chrome not found", "path", chromePath)
		return EmptyProcessInfo()
	}

	// Get window size from global.guisize (format: "WIDTHxHEIGHT" or named size)
	guisize, err := GetParam("global.guisize")
	if err != nil {
		guisize = "800x1280" // default size
	}

	// Track if this is the "palette" size for special positioning
	isPaletteSize := guisize == "palette"

	// Convert named sizes to actual dimensions
	switch guisize {
	case "small":
		guisize = "600x800"
	case "medium":
		guisize = "600x960"
	case "palette":
		guisize = "800x1280"
	case "large":
		guisize = "1024x1600"
	case "fullscreen", "full":
		guisize = "1920x1080"
	}

	// Determine window position
	// For "palette" size, try to place on left monitor if one exists
	windowPosX := 0
	windowPosY := 0
	useKiosk := false
	if isPaletteSize {
		if leftX, leftY, found := GetLeftMonitorPosition(); found {
			windowPosX = leftX
			windowPosY = leftY
			useKiosk = true
			LogInfo("GuiProcessInfo: using left monitor position with kiosk mode", "x", windowPosX, "y", windowPosY)
		}
	}

	// Use a dedicated Chrome profile in the config directory
	chromeDataDir := filepath.Join(ConfigDir(), "chrome")
	if _, err := os.Stat(chromeDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(chromeDataDir, 0755); err != nil {
			LogWarn("Failed to create Chrome data directory", "path", chromeDataDir, "err", err)
		}
	}

	// Clear any cached window placement from Chrome profile so --window-size is respected
	clearChromeWindowPlacement(chromeDataDir)

	// Build Chrome arguments
	// Quote the data dir path since it may contain spaces
	// Use --window-position to help Chrome respect window-size (prevents restoring cached bounds)
	// Use --force-device-scale-factor=1 to prevent Windows DPI scaling from affecting window size
	windowSize := strings.ReplaceAll(guisize, "x", ",")
	kioskFlag := ""
	if useKiosk {
		kioskFlag = "--kiosk "
	}
	args := fmt.Sprintf("--user-data-dir=\"%s\" --profile-directory=Space %s--window-size=%s --window-position=%d,%d --force-device-scale-factor=1 --app=http://127.0.0.1:3330/",
		chromeDataDir, kioskFlag, windowSize, windowPosX, windowPosY)

	LogInfo("GuiProcessInfo", "chromePath", chromePath, "windowSize", windowSize, "args", args)

	return NewProcessInfo("chrome.exe", chromePath, args, nil)
}

// clearChromeWindowPlacement removes cached window placement from Chrome's profile
// so that --window-size is respected on launch
func clearChromeWindowPlacement(chromeDataDir string) {
	// Delete the Preferences file entirely - this ensures Chrome respects --window-size
	// Chrome will recreate it with default settings
	prefsPath := filepath.Join(chromeDataDir, "Space", "Preferences")
	if err := os.Remove(prefsPath); err != nil && !os.IsNotExist(err) {
		LogWarn("Failed to remove Chrome Preferences", "path", prefsPath, "err", err)
	}

	// Also delete Local State which can cache window bounds for apps
	localStatePath := filepath.Join(chromeDataDir, "Local State")
	if err := os.Remove(localStatePath); err != nil && !os.IsNotExist(err) {
		LogWarn("Failed to remove Chrome Local State", "path", localStatePath, "err", err)
	}

}

func ChatProcessInfo() *ProcessInfo {
	fullpath := filepath.Join(paletteRoot, "bin", "palette_chat.exe")
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("No chat executable found, looking for", "path", fullpath)
		return EmptyProcessInfo()
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
			return EmptyProcessInfo()
		}
	}
	exe := filepath.Base(fullpath)
	return NewProcessInfo(exe, fullpath, "", nil)
}
*/

func MmttProcessInfo() *ProcessInfo {

	// The value of PALETTE_MMTT environment variable can be "kinect" or (someday) "oak"
	mmtt := os.Getenv("PALETTE_MMTT")
	if mmtt == "" || mmtt == "none" {
		return EmptyProcessInfo()
	}
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !FileExists(fullpath) {
		LogWarn("mmtt executable not found, looking for", "fullpath", fullpath)
		return EmptyProcessInfo()
	}
	return NewProcessInfo("mmtt_"+mmtt+".exe", fullpath, "", nil)
}
