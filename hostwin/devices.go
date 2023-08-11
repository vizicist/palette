package hostwin

import (
	"encoding/json"
	"fmt"
	"os"
)

// MIDIDeviceEvent is a single MIDI event
type MIDIDeviceEvent struct {
	Timestamp int64 // milliseconds
	Status    int64
	Data1     int64
	Data2     int64
}

// MorphDefs xxx
var MorphDefs map[string]string

// LoadMorphs initializes the list of morphs
func LoadMorphs() error {

	MorphDefs = make(map[string]string)

	// If you have more than one morph, or
	// want the patch assignment to NOT be
	// automatic, put them in here.
	path := ConfigFilePath("morphs.json")
	if !fileExists(path) {
		return fmt.Errorf("unable to get path to morphs.json")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil // It's okay if file isn't present
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s, err=%s", path, err)
	}
	toplevel := f.(map[string]any)

	for serialnum, patchinfo := range toplevel {
		patchname := patchinfo.(string)
		MorphDefs[serialnum] = patchname
	}
	return nil
}
