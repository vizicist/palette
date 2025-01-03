//go:build windows
// +build windows

package kit

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"syscall"
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

// StartExecutable executes something.  If background is true, it doesn't block
func StartExecutableLogOutput(logName string, fullexe string, args ...string) error {
	logWriter := NewExecutableLogWriter(logName)
	_, err := StartExecutableInBackground(fullexe, logWriter, logWriter, args...)
	return err
}

func StartExecutableAndWait(executable string, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {
	return startExecutable(executable, false, stdout, stderr, args...)
}

func StartExecutableInBackground(executable string, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {
	return startExecutable(executable, true, stdout, stderr, args...)
}

// StartExecutable executes something.  If background is true, it doesn't block
func startExecutable(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {

	var cmd *exec.Cmd
	if len(args) == 0 || args[0] == "" {
		cmd = exec.Command(executable)
	} else {
		cmd = exec.Command(executable, args...)
	}

	// This is done so that ctrl-C doesn't kill things
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setpgid:       true,  // for non-Windows systems?
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
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

type gatherWriter struct {
	gathered string
}

func (writer *gatherWriter) Write(bytes []byte) (int, error) {
	writer.gathered += string(bytes)
	return len(bytes), nil
}

func IsRunningExecutable(exe string) (bool,error) {
	// os.Stdout.WriteString("IsRunningExecutable exe=" + exe + "\n")
	stdout := &gatherWriter{}
	stderr := &NoWriter{}
	cmd, err := StartExecutableAndWait("c:\\windows\\system32\\tasklist.exe", stdout, stderr)
	if err != nil {
		LogWarn("IsRunningExecutable of tasklist.exe", "err", err)
		// Assume it's running, to avoid multiple instances
		return true, err
	}
	_ = cmd.Wait() // ignore "Wait was already called"

	scanner := bufio.NewScanner(strings.NewReader(stdout.gathered))
	scanner.Split(bufio.ScanLines)
	exe = isolateExe(exe)
	for scanner.Scan() {
		line := scanner.Text()
		// os.Stdout.WriteString("Scanning line=" + line + "\n")
		words := strings.Fields(line)
		if len(words) > 0 {
			// os.Stdout.WriteString("words0=" + words[0] + " exe=" + exe + "\n")
			if strings.ToLower(words[0]) == exe {
				// os.Stdout.WriteString("IsRunningExecutable " + exe + " returning true\n")
				LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "true")
				return true, nil
			}
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
