//go:build windows
// +build windows

package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"syscall"
)

func StopExecutable(executable string) {
	stdout := &NoWriter{}
	stderr := &NoWriter{}
	cmd, err := StartExecutable("c:\\windows\\system32\\taskkill.exe", false, stdout, stderr, "/F", "/IM", executable)
	if err != nil {
		log.Printf("StopExecutable: err=%s\n", err)
	} else {
		cmd.Wait()
	}
}

// StartExecutable executes something.  If background is true, it doesn't block
func StartExecutableRedirectOutput(cmd string, fullexe string, background bool, args ...string) (*exec.Cmd, error) {
	stdoutWriter := MakeFileWriter(LogFilePath(cmd + ".stdout"))
	if stdoutWriter == nil {
		stdoutWriter = &NoWriter{}
	}
	stderrWriter := MakeFileWriter(LogFilePath(cmd + ".stderr"))
	if stderrWriter == nil {
		stderrWriter = &NoWriter{}
	}
	return StartExecutable(fullexe, true, stdoutWriter, stderrWriter, args...)
}

// StartExecutable executes something.  If background is true, it doesn't block
func StartExecutable(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) (*exec.Cmd, error) {

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

// MorphDefs xxx
var MorphDefs map[string]string

// LoadMorphs initializes the list of morphs
func LoadMorphs() error {

	MorphDefs = make(map[string]string)

	// If you have more than one morph, or
	// want the region assignment to NOT be
	// automatice, put them in here.
	path := LocalConfigFilePath("morphs.json")
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil // It's okay if file isn't present
	}
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s, err=%s", path, err)
	}
	toplevel := f.(map[string]interface{})

	for serialnum, regioninfo := range toplevel {
		regionname := regioninfo.(string)
		if Debug.Morph {
			log.Printf("Setting Morph serial=%s region=%s\n", serialnum, regionname)
		}
		MorphDefs[serialnum] = regionname
		// TheRouter().setRegionForMorph(serialnum, regionname)
	}
	return nil
}

// RealStartCursorInput starts anything needed to provide device inputs
func RealStartCursorInput(callback CursorDeviceCallbackFunc) {

	err := LoadMorphs()
	if err != nil {
		log.Printf("StartDeviceInput: LoadMorphs err=%s\n", err)
	}

	go StartMorph(callback, 1.0)
}

// KillProcess kills a process (synchronously)
func KillProcess(exe string) {
	cmd, err := StartExecutable("cmd.exe", false, NoWriterInstance, NoWriterInstance, "/c", "taskkill", "/f", "/im", exe)
	if err != nil {
		log.Printf("KillProcess: err=%s\n", err)
	} else {
		cmd.Wait()
	}
}
