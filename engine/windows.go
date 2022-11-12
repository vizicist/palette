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

func KillExecutable(executable string) {
	Info("KillExecutable", "executable", executable)
	stdout := &NoWriter{}
	stderr := &NoWriter{}
	cmd, _ := startExecutable("c:\\windows\\system32\\taskkill.exe", false, stdout, stderr, "/F", "/IM", executable)
	// We don't want to complain if the executable isn't
	// currently running, so ignore errors
	cmd.Wait()
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
	_, err := startExecutable(fullexe, true, stdoutWriter, stderrWriter, args...)
	return err
}

// StartExecutable executes something.  If background is true, it doesn't block
func startExecutable(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {

	Info("startExecutable", "executable", executable)

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

func IsRunningExecutable(exe string) bool {
	stdout := &gatherWriter{}
	stderr := &NoWriter{}
	cmd, err := startExecutable("c:\\windows\\system32\\tasklist.exe", false, stdout, stderr)
	if err != nil {
		Warn("IsRunningExecutable tasklist.exe", "err", err)
		return false
	}
	cmd.Wait()

	scanner := bufio.NewScanner(strings.NewReader(stdout.gathered))
	scanner.Split(bufio.ScanLines)
	exe = strings.ToLower(exe)
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 0 {
			if strings.ToLower(words[0]) == exe {
				return true
			}
		}
	}
	return false
}
