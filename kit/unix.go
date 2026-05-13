//go:build !windows

package kit

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// Spawn executes something.  If background is true, it doesn't block
func Spawn(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) error {

	cmd := exec.Command(executable, args...)

	// This is done so that ctrl-C doesn't kill things
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // for non-Windows systems?
		// CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
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
		// Don't do Fatal here - some commands like taskkill will fail harmlessly
		LogIfError(err)
		return err
	}
	LogInfo("Spawn", "bg", background, "cmd")
	return nil
}

var noWriter = &NoWriter{}

// StartDeviceInput starts anything needed to provide device inputs
func StartDeviceInput() {
}

// KillProcess kills a process (synchronously)
func KillExecutable(exe string) {
	exe = isolateExe(exe)
	if exe == "" {
		LogWarn("KillExecutable: empty name?")
		return
	}
	LogInfo("KillExecutable", "exe", exe)
	for _, candidate := range executableNameCandidates(exe) {
		_ = exec.Command("pkill", "-x", candidate).Run()
	}
}

func IsSamplesplitterRunning() (bool, error) {
	err := exec.Command("pgrep", "-f", "samplesplitter.py").Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func KillSamplesplitter() {
	_ = exec.Command("pkill", "-f", "samplesplitter.py").Run()
}

// KillByPid kills a process by its PID
func KillByPid(pid int) {
	if pid <= 0 {
		return
	}
	LogWarn("KillByPid in unix.go not tested", "pid", pid)
	Spawn("kill", false, noWriter, noWriter, "-9", fmt.Sprintf("%d", pid))
}

func FindWindowByTitleContains(substring string) bool {
	return false
}

func CloseWindowByTitle(title string) bool {
	if runtime.GOOS == "darwin" {
		if title == "Palette Control" {
			closed := closeMacChromeWindowByTitle(title)
			killed := killPaletteChromeProfile()
			return closed || killed
		}
		if closeMacChromeWindowByTitle(title) {
			return true
		}
	}
	return false
}

func GetLeftMonitorPosition() (x int, y int, found bool) {
	return 0, 0, false
}

func IsRunningExecutable(exe string) (bool, error) {
	exe = isolateExe(exe)
	if exe == "" {
		return false, fmt.Errorf("IsRunningExecutable: empty executable name")
	}
	err := exec.Command("pgrep", "-x", exe).Run()
	if err == nil {
		LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "true")
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "false")
		return false, nil
	}
	return false, err
}

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

func startExecutable(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {
	cmdArgs := normalizeProcessArgs(args...)
	cmd := exec.Command(executable, cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var err error
	if background {
		err = cmd.Start()
	} else {
		err = cmd.Run()
	}
	return cmd, err
}

func normalizeProcessArgs(args ...string) []string {
	if len(args) == 0 {
		return nil
	}
	if len(args) == 1 {
		if args[0] == "" {
			return nil
		}
		return splitCommandLine(args[0])
	}
	return args
}

func splitCommandLine(line string) []string {
	args := []string{}
	var b strings.Builder
	inQuote := false
	escaped := false

	for _, r := range line {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			inQuote = !inQuote
		case (r == ' ' || r == '\t') && !inQuote:
			if b.Len() > 0 {
				args = append(args, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	if b.Len() > 0 {
		args = append(args, b.String())
	}
	return args
}

func isolateExe(exepath string) string {
	exepath = strings.ReplaceAll(exepath, "\\", "/")
	exe := filepath.Base(exepath)
	if exe == "." || exe == "/" {
		return exepath
	}
	return exe
}

func executableNameCandidates(exe string) []string {
	candidates := []string{exe}
	if exe != "" {
		candidates = append(candidates, strings.ToLower(exe))
		candidates = append(candidates, strings.ToUpper(exe[:1])+exe[1:])
	}
	seen := map[string]bool{}
	unique := []string{}
	for _, candidate := range candidates {
		if candidate != "" && !seen[candidate] {
			seen[candidate] = true
			unique = append(unique, candidate)
		}
	}
	return unique
}

func closeMacChromeWindowByTitle(title string) bool {
	script := fmt.Sprintf(`
tell application "Google Chrome"
	set closedAny to false
	repeat with w in windows
		if name of w contains %q then
			close w
			set closedAny to true
		end if
	end repeat
	return closedAny
end tell`, title)
	err := exec.Command("osascript", "-e", script).Run()
	return err == nil
}

func killPaletteChromeProfile() bool {
	chromeDataDir := filepath.Join(ConfigDir(), "chrome")
	err := exec.Command("pkill", "-f", chromeDataDir).Run()
	if err == nil {
		return true
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false
	}
	LogIfError(err)
	return false
}
