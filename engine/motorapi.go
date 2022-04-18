package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

// ExecuteAPI xxx
func (motor *Motor) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	if Debug.MotorAPI {
		log.Printf("MotorAPI: api=%s rawargs=%s\n", api, rawargs)
	}

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if strings.HasPrefix(api, "visual.") {
		msg := osc.NewMessage("/api")
		msg.Append(strings.TrimPrefix(api, "visual."))
		msg.Append(rawargs)
		motor.toFreeFramePluginForLayer(msg)
	}

	switch api {

	case "send":
		log.Printf("API send should be sending all parameters for region=%s\n", motor.padName)
		motor.sendAllParameters()
		return "", err

	case "loop_recording":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			motor.loopIsRecording = v
		} else {
			err = e
		}

	case "loop_playing":
		v, e := needBoolArg("onoff", api, args)
		if e == nil && v != motor.loopIsPlaying {
			motor.loopIsPlaying = v
			motor.terminateActiveNotes()
		} else {
			err = e
		}

	case "loop_clear":
		motor.loop.Clear()
		motor.clearGraphics()
		motor.sendANO()

	case "loop_comb":
		motor.loopComb()

	case "loop_length":
		i, e := needIntArg("value", api, args)
		if e == nil {
			nclicks := Clicks(i)
			if nclicks != motor.loop.length {
				motor.loop.SetLength(nclicks)
			}
		} else {
			err = e
		}

	case "loop_fade":
		f, e := needFloatArg("fade", api, args)
		if e == nil {
			motor.fadeLoop = f
		} else {
			err = e
		}

	case "ANO":
		motor.sendANO()

	case "midi_thru":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			motor.MIDIThru = v
		} else {
			err = e
		}

	case "midi_setscale":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			motor.MIDISetScale = v
		} else {
			err = e
		}

	case "midi_usescale":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			motor.MIDIUseScale = v
		} else {
			err = e
		}

	case "clearexternalscale":
		motor.clearExternalScale()
		motor.MIDINumDown = 0

	case "midi_quantized":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			motor.MIDIQuantized = v
		}

	case "midi_thruscadjust":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			motor.MIDIThruScadjust = v
		} else {
			err = e
		}

	case "set_transpose":
		v, e := needIntArg("value", api, args)
		if e == nil {
			motor.TransposePitch = v
			if Debug.Transpose {
				log.Printf("motor API set_transpose TransposePitch=%v", v)
			}
		} else {
			err = e
		}

	default:
		err = fmt.Errorf("Motor.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}

func (motor *Motor) sendAllParameters() {
	for nm := range motor.params.values {
		val, e := motor.params.paramValueAsString(nm)
		if e != nil {
			log.Printf("Unexepected error from paramValueAsString for nm=%s\n", nm)
			// Keep going
			continue
		}
		// This assumes that if you set a parameter to the same value,
		// that it will re-send the mesasges to Resolume for visual.* params
		err := motor.SetOneParamValue(nm, val)
		if err != nil {
			log.Printf("sendAllParameters nm=%s val=%s err=%s\n", nm, val, err)
		}
	}
}

func (motor *Motor) loadPreset(preset string) error {

	path := ReadablePresetFilePath(preset)
	log.Printf("loadPreset region=%s path=%s\n", motor.padName, path)
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		return err
	}

	/*
		// If the preset value is of the form {category}.{preset},
		// then we pull off the category and add it as a prefix
		// to the parameter names.
		prefix := ""
		i := strings.Index(preset, ".")
		if i >= 0 {
			prefix = preset[0 : i+1]
		}
		// HOWEVER, snap.* presets already have
		// category prefixes on the parameter names,
		// so we don't need to add them.
		if prefix == "snap." || prefix == "quad." {
			prefix = ""
		}
	*/

	for nm, ival := range paramsmap {
		val, okval := ival.(string)
		if !okval {
			log.Printf("nm=%s value isn't a string in params json", nm)
		}
		// fullname := prefix + nm
		fullname := nm
		// This is where the parameter values get applied,
		// which may trigger things (like sending OSC)
		err = motor.SetOneParamValue(fullname, val)
		if err != nil {
			log.Printf("Loading preset %s, err=%s\n", preset, err)
			// Don't abort the whole load, i.e. we are tolerant
			// of unknown parameters in the preset
		}
	}
	return nil
}

func (motor *Motor) restoreCurrentSnap() {
	motor.loadPreset("snap._Current_" + motor.padName)
}

func (motor *Motor) saveCurrentAsPreset(preset string) error {
	path := WriteablePresetFilePath(preset)
	return motor.saveCurrentSnapInPath(path)
}

func (motor *Motor) saveCurrentSnap() error {
	return motor.saveCurrentAsPreset("snap._Current_" + motor.padName)
}

func (motor *Motor) saveCurrentSnapInPath(path string) error {

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := motor.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		valstring, e := motor.params.paramValueAsString(fullName)
		if e != nil {
			log.Printf("Unexepected error from paramValueAsString for nm=%s\n", fullName)
			continue
		}
		s += fmt.Sprintf("%s        \"%s\":\"%s\"", sep, fullName, valstring)
		sep = ",\n"
	}
	s += "\n    }\n}"
	data := []byte(s)
	// log.Printf("SaveCurrentSnapInPath %s path=%s\n", motor.padName, path)
	return ioutil.WriteFile(path, data, 0644)
}

func LoadParamsMap(path string) (map[string]interface{}, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshal path=%s, err=%s", path, err)
	}
	toplevel, ok := f.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to convert params to map[string]interface{}")

	}
	params, okparams := toplevel["params"]
	if !okparams {
		return nil, fmt.Errorf("no params value in json")
	}
	paramsmap, okmap := params.(map[string]interface{})
	if !okmap {
		return nil, fmt.Errorf("params value is not a map[string]string in jsom")
	}
	return paramsmap, nil
}
