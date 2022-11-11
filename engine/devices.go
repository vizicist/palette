package engine

import (
	"encoding/json"
	"fmt"
	"log"
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
	// want the player assignment to NOT be
	// automatice, put them in here.
	path := ConfigFilePath("morphs.json")
	if !fileExists(path) {
		return fmt.Errorf("unable to get path to morphs.json")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil // It's okay if file isn't present
	}
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s, err=%s", path, err)
	}
	toplevel := f.(map[string]interface{})

	for serialnum, playerinfo := range toplevel {
		playername := playerinfo.(string)
		if Debug.Morph {
			log.Printf("Setting Morph serial=%s player=%s\n", serialnum, playername)
		}
		MorphDefs[serialnum] = playername
		// TheRouter().setPlayerForMorph(serialnum, playername)
	}
	return nil
}
