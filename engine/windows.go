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

// Spawn executes something.  If background is true, it doesn't block
func Spawn(executable string, background bool, stdout io.Writer, stderr io.Writer, args ...string) error {

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
	if err != nil {
		// Don't do Fatal here - some commands like taskkill will fail harmlessly
		log.Printf("Spawn: cmd=%s err=%s\n", cmd, err)
		return err
	}
	log.Printf("Spawn: bg=%v cmd=%s\n", background, cmd)
	return nil
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
		if DebugUtil.Morph {
			log.Printf("Setting Morph serial=%s region=%s\n", serialnum, regionname)
		}
		MorphDefs[serialnum] = regionname
		TheRouter().setRegionForMorph(serialnum, regionname)
	}
	return nil
}

var noWriter = &NoWriter{}

// RealStartGestureInput starts anything needed to provide device inputs
func RealStartGestureInput(callback GestureDeviceCallbackFunc) {

	err := LoadMorphs()
	if err != nil {
		log.Printf("StartDeviceInput: LoadMorphs err=%s\n", err)
	}

	go StartMorph(callback, 1.0)

	return
}

// KillProcess kills a process (synchronously)
func KillProcess(exe string) {
	Spawn("cmd.exe", false, noWriter, noWriter, "/c", "taskkill", "/f", "/im", exe)

}
