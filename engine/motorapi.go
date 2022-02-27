package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

	case "load":
		preset, okpreset := args["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		log.Printf("Should be loading preset=%s\n", preset)
		path := ReadablePresetFilePath(preset)
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		var f interface{}
		err = json.Unmarshal(bytes, &f)
		if err != nil {
			return "", fmt.Errorf("unable to Unmarshal path=%s, err=%s", path, err)
		}
		toplevel := f.(map[string]interface{})
		params, okparams := toplevel["params"]
		if !okparams {
			return "", fmt.Errorf("no params value in jsom")
		}
		paramsmap, okmap := params.(map[string]interface{})
		if !okmap {
			return "", fmt.Errorf("params value is not a map in jsom")
		}
		// If the preset value is of the form {category}.{preset},
		// then we pull off the category and add it as a prefix
		// to the parameter names.
		prefix := ""
		i := strings.Index(preset, ".")
		if i >= 0 {
			prefix = preset[0 : i+1]
		}
		for nm := range paramsmap {
			i := paramsmap[nm]
			s, oks := i.(string)
			if !oks {
				log.Printf("nm=%s value isn't a string in params json", nm)
			}
			log.Printf("paramsmap nm=%s s=%s\n", nm, s)
			fullname := prefix + nm
			err = motor.SetOneParamValue(fullname, s)
			if err != nil {
				return "", err
			}
		}
		return "", nil

	case "save":
		preset, okpreset := args["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		log.Printf("Should be saving preset=%s\n", preset)
		wantCategory, _ := PresetNameSplit(preset)
		path := WriteablePresetFilePath(preset)
		s := ""
		sep := "{\n    \"params\": {\n"
		for nm := range motor.params.values {
			thisCategory, thisPreset := PresetNameSplit(nm)
			if thisCategory != wantCategory {
				continue
			}
			valstring, e := motor.params.paramValueAsString(nm)
			if e != nil {
				log.Printf("Unexepected error from paramValueAsString for nm=%s\n", nm)
				continue
			}
			s += fmt.Sprintf("%s        \"%s\":\"%s\"", sep, thisPreset, valstring)
			sep = ",\n"
		}
		s += "\n    }\n}"
		data := []byte(s)
		err := ioutil.WriteFile(path, data, 0644)
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
		i, e := needIntArg("length", api, args)
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
