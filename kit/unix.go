//go:build !windows

package kit

import (
	"fmt"
	"io"
	"os/exec"
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
	LogWarn("KillProcess in unix.go not tested", "exe", exe)
	Spawn("pkill", false, noWriter, noWriter, exe)
}

func IsRunningExecutable(exe string) (bool, error) {
	err := fmt.Errorf("unix.go: IsRunningExecutable needs work")
	return false, err
}

func StartExecutableLogOutput(logName string, fullexe string, args ...string) error {
	err := fmt.Errorf("unix.go: StartExecutableLogOutput needs work")
	// logWriter := NewExecutableLogWriter(logName)
	// _, err := StartExecutableInBackground(fullexe, logWriter, logWriter, args...)
	return err
}
