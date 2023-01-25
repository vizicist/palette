//go:build windows
// +build windows

package engine

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

func killExecutable(executable string) error {
	exe := isolateExe(executable)
	// os.Stderr.WriteString("KillExecutable " + executable + "\n")
	LogInfo("KillExecutable", "executable", executable, "exe", exe)
	// NOTE: do NOT use taskkill.exe instead of taskkill, it doesn't work.
	cmd := exec.Command("taskkill", "/F", "/IM", exe)
	// os.Stderr.WriteString(fmt.Sprintf("cmd = %#v\n", cmd))
	return cmd.Run()
}

// StartExecutable executes something.  If background is true, it doesn't block
func StartExecutableLogOutput(logName string, fullexe string, background bool, args ...string) error {
	stdoutWriter := MakeFileWriter(LogFilePath(logName + ".stdout"))
	if stdoutWriter == nil {
		stdoutWriter = &NoWriter{}
	}
	stderrWriter := MakeFileWriter(LogFilePath(logName + ".stderr"))
	if stderrWriter == nil {
		stderrWriter = &NoWriter{}
	}
	LogInfo("startExecutable", "executable", fullexe)
	_, err := startExecutable(fullexe, true, stdoutWriter, stderrWriter, args...)
	return err
}

// StartExecutable executes something.  If background is true, it doesn't block
func startExecutable(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {

	cmd := exec.Command(executable, args...)

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
	return cmd, err
}

type gatherWriter struct {
	gathered string
}

func (writer *gatherWriter) Write(bytes []byte) (int, error) {
	writer.gathered += string(bytes)
	return len(bytes), nil
}

func isRunningExecutable(exe string) bool {
	// os.Stdout.WriteString("IsRunningExecutable exe=" + exe + "\n")
	stdout := &gatherWriter{}
	stderr := &NoWriter{}
	cmd, err := startExecutable("c:\\windows\\system32\\tasklist.exe", false, stdout, stderr)
	if err != nil {
		LogWarn("IsRunningExecutable tasklist.exe", "err", err)
		LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "true")
		return false
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
				return true
			}
		}
	}
	LogOfType("process", "IsRunningExecutable", "exe", exe, "returning", "false")
	return false
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
