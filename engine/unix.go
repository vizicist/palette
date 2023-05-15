//go:build !windows
// +build !windows

package engine

import (
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
	Info("Spawn", "bg", background, "cmd")
	return nil
}

var noWriter = &NoWriter{}

// StartDeviceInput starts anything needed to provide device inputs
func StartDeviceInput() {
	return
}

// KillProcess kills a process (synchronously)
func KillProcess(exe string) {
	Warn("KillProcess in unix.go not tested", "exe", exe)
	Spawn("pkill", false, noWriter, noWriter, exe)
}
