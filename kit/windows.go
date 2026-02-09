//go:build windows

package kit

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

func KillExecutable(executable string) {
	exe := isolateExe(executable)
	if executable == "" {
		LogWarn("KillExecutable: empty name?")
	}
	LogInfo("KillExecutable", "executable", executable, "exe", exe)
	// NOTE: do NOT use taskkill.exe instead of taskkill, it doesn't work.
	cmd := exec.Command("taskkill", "/F", "/IM", exe)
	// If the process isn't currently running,
	// cmd.Run gives an "exit status 128" error,
	// and doesn't seem like there's any other useful errors,
	// so just ignore all errors.
	_ = cmd.Run()
}

// KillByPid kills a process by its PID, including child processes
func KillByPid(pid int) {
	if pid <= 0 {
		return
	}
	LogInfo("KillByPid", "pid", pid)
	// /T kills child processes too
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	_ = cmd.Run()
}

// StartExecutable executes something.  If background is true, it doesn't block
// Returns the PID of the started process
func StartExecutableLogOutput(logName string, fullexe string, args ...string) (int, error) {
	logWriter := NewExecutableLogWriter(logName)
	cmd, err := StartExecutableInBackground(fullexe, logWriter, logWriter, args...)
	if err != nil {
		return 0, err
	}
	if cmd.Process != nil {
		return cmd.Process.Pid, nil
	}
	return 0, nil
}

func StartExecutableAndWait(executable string, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {
	return startExecutable(executable, false, stdout, stderr, args...)
}

func StartExecutableInBackground(executable string, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {
	return startExecutable(executable, true, stdout, stderr, args...)
}

// quoteArg quotes a path argument if it contains spaces.
// Args containing "--" are treated as pre-formatted command line fragments and not quoted.
func quoteArg(arg string) string {
	// If arg contains "--", it's a pre-formatted command line with flags - don't quote
	if strings.Contains(arg, "--") {
		return arg
	}
	// If arg contains spaces and isn't already quoted, quote it
	if strings.Contains(arg, " ") && !strings.HasPrefix(arg, `"`) {
		return `"` + arg + `"`
	}
	return arg
}

// StartExecutable executes something.  If background is true, it doesn't block
func startExecutable(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {

	var cmd *exec.Cmd
	if len(args) == 0 || args[0] == "" {
		cmd = exec.Command(executable)
	} else {
		cmd = exec.Command(executable)
	}

	// This is done so that ctrl-C doesn't kill things
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setpgid:       true,  // for non-Windows systems?
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	// If we have arguments, use CmdLine to pass them as a raw command line string
	// This properly handles paths with spaces without Go's escaping interfering
	if len(args) > 0 && args[0] != "" {
		// Quote each argument that contains spaces
		quotedArgs := make([]string, len(args))
		for i, arg := range args {
			quotedArgs[i] = quoteArg(arg)
		}
		cmdLine := `"` + executable + `" ` + strings.Join(quotedArgs, " ")
		cmd.SysProcAttr.CmdLine = cmdLine
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var err error
	if background {
		err = cmd.Start()
	} else {
		err = cmd.Run()
	}
	if err != nil {
		LogInfo("err in StartExecutable")
	}
	return cmd, err
}

// IsRunningExecutable checks if an executable is running using Windows API
func IsRunningExecutable(exe string) (bool, error) {
	exe = isolateExe(exe)

	// Create a snapshot of all processes
	snapshot, _, err := createToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == INVALID_HANDLE_VALUE {
		LogWarn("IsRunningExecutable: CreateToolhelp32Snapshot failed", "err", err)
		// Assume it's running to avoid multiple instances
		return true, fmt.Errorf("CreateToolhelp32Snapshot failed: %v", err)
	}
	defer closeHandle.Call(snapshot)

	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	// Get the first process
	ret, _, _ := process32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "false")
		return false, nil
	}

	for {
		// Convert the exe name from UTF-16 to string
		processName := syscall.UTF16ToString(entry.ExeFile[:])
		if strings.ToLower(processName) == exe {
			LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "true")
			return true, nil
		}

		// Get the next process
		ret, _, _ = process32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "false")
	return false, nil
}

func isolateExe(exepath string) string {
	exepath = strings.ToLower(exepath)
	exepath = strings.ReplaceAll(exepath, "\\", "/")
	exeparts := strings.Split(exepath, "/")
	exe := exepath
	if len(exeparts) > 0 {
		exe = exeparts[len(exeparts)-1]
	}
	return exe
}

// Windows API types for monitor enumeration
type rect struct {
	Left, Top, Right, Bottom int32
}

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

// Windows API types for process enumeration
// PROCESSENTRY32W structure for Process32FirstW/Process32NextW
type processEntry32W struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriClassBase      int32
	Flags             uint32
	ExeFile           [260]uint16 // MAX_PATH = 260
}

// Windows API constants for process functions
const (
	TH32CS_SNAPPROCESS = 0x00000002
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	INVALID_HANDLE_VALUE = ^uintptr(0)
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	enumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	getMonitorInfoW     = user32.NewProc("GetMonitorInfoW")

	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	createToolhelp32Snapshot  = kernel32.NewProc("CreateToolhelp32Snapshot")
	process32FirstW           = kernel32.NewProc("Process32FirstW")
	process32NextW            = kernel32.NewProc("Process32NextW")
	openProcess               = kernel32.NewProc("OpenProcess")
	closeHandle               = kernel32.NewProc("CloseHandle")
)

// monitorBounds collects monitor positions during enumeration
var monitorBounds []rect

// monitorEnumProc is the callback for EnumDisplayMonitors
func monitorEnumProc(hMonitor uintptr, hdcMonitor uintptr, lprcMonitor uintptr, dwData uintptr) uintptr {
	var mi monitorInfo
	mi.Size = uint32(unsafe.Sizeof(mi))
	ret, _, _ := getMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if ret != 0 {
		monitorBounds = append(monitorBounds, mi.Monitor)
	}
	return 1 // Continue enumeration
}

// GetLeftMonitorPosition returns the x,y position of a monitor to the left of the primary monitor.
// If there's no left monitor or detection fails, returns 0,0,false.
// If a left monitor exists, returns its x,y position and true.
func GetLeftMonitorPosition() (x int, y int, found bool) {
	// Clear previous results
	monitorBounds = nil

	// Enumerate all monitors using Windows API
	callback := syscall.NewCallback(monitorEnumProc)
	ret, _, err := enumDisplayMonitors.Call(0, 0, callback, 0)
	if ret == 0 {
		LogWarn("GetLeftMonitorPosition: EnumDisplayMonitors failed", "err", err)
		return 0, 0, false
	}

	// Find monitor with negative X (to the left of primary which is at 0,0)
	leftMostX := int32(0)
	leftMostY := int32(0)
	hasLeftMonitor := false

	for _, bounds := range monitorBounds {
		if bounds.Left < 0 {
			if !hasLeftMonitor || bounds.Left < leftMostX {
				leftMostX = bounds.Left
				leftMostY = bounds.Top
				hasLeftMonitor = true
			}
		}
	}

	if hasLeftMonitor {
		LogInfo("GetLeftMonitorPosition: found left monitor", "x", leftMostX, "y", leftMostY)
	}
	return int(leftMostX), int(leftMostY), hasLeftMonitor
}

// IsPidRunning checks if a specific process ID is still running using Windows API
func IsPidRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Try to open the process with minimal access rights
	// If the process exists, OpenProcess will succeed
	handle, _, _ := openProcess.Call(
		PROCESS_QUERY_LIMITED_INFORMATION,
		0,
		uintptr(pid),
	)

	if handle == 0 {
		// Process doesn't exist or we don't have access
		return false
	}

	// Process exists, close the handle
	closeHandle.Call(handle)
	return true
}
