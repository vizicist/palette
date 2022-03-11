package engine

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExecuteAPI xxx
func (r *Router) ExecuteAPI(api string, nuid string, rawargs string) (result interface{}, err error) {

	apiargs, e := StringMap(rawargs)
	if e != nil {
		result = ErrorResponse(fmt.Errorf("Router.ExecuteAPI: Unable to interpret value - %s", rawargs))
		log.Printf("Router.ExecuteAPI: bad rawargs value = %s\n", rawargs)
		return result, e
	}

	result = "" // pre-populate most common result

	switch api {

	// These are the APIs that are region-specific.
	// "load" and "save" have one exception - they
	// are not region-specific when used on quad.* presets.
	case "load", "save", "get", "set",
		"loop_comb", "loop_set", "loop_clear",
		"loop_length", "loop_fade",
		"loop_recording", "loop_playing",
		"quant", "scale", "vol", "comb",
		"midi_thru", "midi_setscale", "midi_usescale",
		"midi_quantized", "midi_thruscadjust",
		"transpose":

		region, regionok := apiargs["region"]
		if !regionok {
			region = "*"
		}
		return r.executeRegionAPI(region, api, apiargs, rawargs)

	case "list":
		return presetList(apiargs)

	case "start", "stop":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("executeProcessAPI: missing process argument")
		} else if api == "start" {
			err = StartRunning(process)
		} else {
			err = StopRunning(process)
		}
		return "", err

	case "activate":
		go resolumeActivate()
		go biduleActivate()
		return "", nil

	case "sendlogs":
		return "", SendLogs()

	default:
		// All other APIs are of the form {apitype}.{api}.
		// So far there's only global.* APIs.
		words := strings.Split(api, ".")
		apitype := words[0]
		apisuffix := ""
		if len(words) > 1 {
			apisuffix = words[1]
		}
		if apitype == "global" {
			return r.executeGlobalAPI(apisuffix, apiargs)
		} else {
			return nil, fmt.Errorf("ExecuteAPI: unknown prefix on api=%s", api)
		}

	}
}

func presetList(apiargs map[string]string) (string, error) {

	wantCategory := optionalStringArg("category", apiargs, "")
	result := "["
	sep := ""

	walker := func(path string, info os.FileInfo, err error) error {
		// log.Printf("Crawling: %#v\n", path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Only look at .json files
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		path = strings.TrimSuffix(path, ".json")
		// the last two components of the path are category and preset
		thisCategory := ""
		thisPreset := ""
		lastslash2 := -1
		lastslash := strings.LastIndex(path, "\\")
		if lastslash >= 0 {
			thisPreset = path[lastslash+1:]
			path2 := path[0:lastslash]
			lastslash2 = strings.LastIndex(path2, "\\")
			if lastslash2 >= 0 {
				thisCategory = path2[lastslash2+1:]
			}
		}
		if wantCategory == "*" || thisCategory == wantCategory {
			result += sep + "\"" + thisCategory + "." + thisPreset + "\""
			sep = ","
		}
		return nil
	}

	presetsDir1 := filepath.Join(PaletteDir(), "presets")
	err := filepath.Walk(presetsDir1, walker)
	if err != nil {
		log.Printf("filepath.Walk: err=%s\n", err)
	}
	presetsDir2 := filepath.Join(LocalPaletteDir(), "presets")
	err = filepath.Walk(presetsDir2, walker)
	if err != nil {
		log.Printf("filepath.Walk: err=%s\n", err)
	}
	result += "]"
	return result, nil
}

/*
func (r *Router) executePresetAPI(api string, apiargs map[string]string, rawargs string) (string, error) {
	switch api {
	case "list":
		return presetList(apiargs)

	default:
		return "", fmt.Errorf("unrecognized prefix sub-command: %s", api)
	}
}
*/

func (r *Router) executeRegionAPI(region string, api string, apiargs map[string]string, rawargs string) (result string, err error) {

	// The "load" and "save" APIs are non-region-specific
	// when loading "quad.*" presets.
	if api == "load" || api == "save" {
		// So, let's see if it's a quad.* preset
		preset, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("no preset argument in load api")
		}
		if strings.HasPrefix(preset, "quad.") {
			if api == "load" {
				log.Printf("Loading preset=%s", preset)
				return "", r.loadQuadPreset(preset)
			} else {
				log.Printf("Saving preset=%s", preset)
				return "", r.saveQuadPreset(preset)
			}
		}
		// otherwise continue to treat it
		// like a normal region-specific api
	}

	switch api {

	// XXX - I'm not sure there's a reason why the "set" api
	// is handled up here instead of down in the motors.
	case "set":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeRegionAPI: missing name argument")
		}
		value, ok := apiargs["value"]
		if !ok {
			return "", fmt.Errorf("executeRegionAPI: missing value argument")
		}
		for thisRegion, motor := range r.motors {
			if region == "*" || region == thisRegion {
				err = motor.SetOneParamValue(name, value)
				if err != nil {
					return "", err
				}
			}
		}
		return "", nil

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeRegionAPI: missing name argument")
		}
		if region == "*" {
			return "", fmt.Errorf("executeRegionAPI: get can't handle *")
		}
		motor, ok := r.motors[region]
		if !ok {
			return "", fmt.Errorf("no region named %s", region)
		}
		return motor.params.paramValueAsString(name)

	case "load":
		for thisRegion, motor := range r.motors {
			if region == "*" || thisRegion == region {
				_, err := motor.ExecuteAPI(api, apiargs, "")
				if err != nil {
					return "", err
				}
			}
		}
		return "", nil

	case "save":
		if region == "*" {
			return "", fmt.Errorf("executeRegionAPI: save can't handle *")
		}
		motor, ok := r.motors[region]
		if !ok {
			return "", fmt.Errorf("ExecuteAPI: no region named %s", region)
		}
		r, err := motor.ExecuteAPI(api, apiargs, rawargs)
		if err != nil {
			return "", err
		}
		if r != "" {
			return "", fmt.Errorf("unexpected non-null result from preset load api")
		}
		return "", nil

	default:
		// The region-specific APIs above are handled
		// here in the Router context, but for everything else,
		// we punt down to the region's motor.
		// region can be A, B, C, D, or *
		for tmpRegion, motor := range r.motors {
			if region == "*" || tmpRegion == region {
				_, err := motor.ExecuteAPI(api, apiargs, rawargs)
				if err != nil {
					return "", err
				}
			}
		}
		return "", nil
	}
}

func (r *Router) saveQuadPreset(preset string) error {

	// wantCategory is sound, visual, effect, snap, or quad
	path := WriteablePresetFilePath(preset)
	s := "{\n    \"params\": {\n"

	sep := ""
	for _, motor := range r.motors {
		// Print the parameter values sorted by name
		fullNames := motor.params.values
		sortedNames := make([]string, 0, len(fullNames))
		for k := range fullNames {
			sortedNames = append(sortedNames, k)
		}
		sort.Strings(sortedNames)

		for _, fullName := range sortedNames {
			valstring, e := motor.params.paramValueAsString(fullName)
			if e != nil {
				log.Printf("Unexepected error from paramValueAsString for nm=%s\n", fullName)
				continue
			}
			s += fmt.Sprintf("%s        \"%s-%s\":\"%s\"", sep, motor.padName, fullName, valstring)
			sep = ",\n"
		}
	}
	s += "\n    }\n}"
	data := []byte(s)
	return ioutil.WriteFile(path, data, 0644)
}

func (r *Router) loadQuadPreset(preset string) error {
	path := ReadablePresetFilePath(preset)
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		return err
	}
	// Here's where the params get applied,
	// which among other things
	// may result in sending OSC messages out.
	for name, ival := range paramsmap {
		value, ok := ival.(string)
		if !ok {
			return fmt.Errorf("value of name=%s isn't a string", name)
		}
		// In a quad file, the parameter names are of the form:
		// {region}-{parametername}
		words := strings.SplitN(name, "-", 2)
		motor, ok := r.motors[words[0]]
		if !ok {
			return fmt.Errorf("no region named %s", words[0])
		}
		// We expect the parameter to be of the form
		// {category}.{parameter}, but old "quad" files
		// didn't include the category.
		parameterName := words[1]
		if !strings.Contains(parameterName, ".") {
			log.Printf("loadQuadPreset: preset=%s parameter=%s is in OLD format, not supported", preset, parameterName)
			return fmt.Errorf("")
		}
		// use words[1] so the motor doesn't see the region name
		err = motor.SetOneParamValue(parameterName, value)
		if err != nil {
			// XXX - might not want to
			// fail completely on individual failures
			return err
		}
	}
	return nil
}

/*
func (r *Router) executeProcessAPI(api string, apiargs map[string]string) (result string, err error) {
	switch api {

	case "start":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("executeProcessAPI: missing process argument")
		} else {
			err = StartRunning(process)
		}

	case "stop":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("executeProcessAPI: missing process argument")
		} else {
			err = StopRunning(process)
		}

	default:
		err = fmt.Errorf("executeProcessAPI: unknown api %s", api)
	}

	if err != nil {
		return "", err
	} else {
		return result, nil
	}
}
*/

func (r *Router) executeGlobalAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "midi_midifile":
		return "", fmt.Errorf("midi_midifile API has been removed")

	case "echo":
		value, ok := apiargs["value"]
		if !ok {
			value = "ECHO!"
		}
		result = value

	case "fakemidi":
		// publish fake event for testing
		me := MIDIDeviceEvent{
			Timestamp: int64(0),
			Status:    0x90,
			Data1:     0x10,
			Data2:     0x10,
		}
		eee := PublishMIDIDeviceEvent(me)
		if eee != nil {
			log.Printf("InputListener: me=%+v err=%s\n", me, eee)
		}
		/*
			d := time.Duration(secs) * time.Second
			log.Printf("hang: d=%+v\n", d)
			time.Sleep(d)
			log.Printf("hang is done\n")
		*/

	case "debug":
		s, err := needStringArg("debug", api, apiargs)
		if err == nil {
			b, err := needBoolArg("onoff", api, apiargs)
			if err == nil {
				setDebug(s, b)
			}
		}

	case "set_tempo_factor":
		v, err := needFloatArg("value", api, apiargs)
		if err == nil {
			ChangeClicksPerSecond(float64(v))
		}

	case "set_transpose":
		v, err := needFloatArg("value", api, apiargs)
		if err == nil {
			for _, motor := range r.motors {
				motor.TransposePitch = int(v)
			}
		}

	case "set_transposeauto":
		b, err := needBoolArg("onoff", api, apiargs)
		if err == nil {
			r.transposeAuto = b
			// Quantizing CurrentClick() to a beat or measure might be nice
			r.transposeNext = CurrentClick() + r.transposeBeats*oneBeat
			for _, motor := range r.motors {
				motor.TransposePitch = 0
			}
		}

	case "set_scale":
		v, err := needStringArg("value", api, apiargs)
		if err == nil {
			for _, motor := range r.motors {
				err = motor.SetOneParamValue("misc.scale", v)
				if err != nil {
					break
				}
			}
		}

	case "audio_reset":
		go r.audioReset()

	case "recordingStart":
		r.recordingOn = true
		if r.recordingFile != nil {
			log.Printf("Hey, recordingFile wasn't nil?\n")
			r.recordingFile.Close()
		}
		r.recordingFile, err = os.Create(recordingsFile("LastRecording.json"))
		if err != nil {
			return "", err
		}
		r.recordingBegun = time.Now()
		if r.recordingOn {
			r.recordEvent("global", "*", "start", "{}")
		}
	case "recordingSave":
		var name string
		name, err = needStringArg("name", api, apiargs)
		if err == nil {
			err = r.recordingSave(name)
		}

	case "recordingStop":
		if r.recordingOn {
			r.recordEvent("global", "*", "stop", "{}")
		}
		if r.recordingFile != nil {
			r.recordingFile.Close()
			r.recordingFile = nil
		}
		r.recordingOn = false

	case "recordingPlay":
		name, err := needStringArg("name", api, apiargs)
		if err == nil {
			events, err := r.recordingLoad(name)
			if err == nil {
				r.sendANO()
				go r.recordingPlayback(events)
			}
		}

	case "recordingPlaybackStop":
		r.recordingPlaybackStop()

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		err = fmt.Errorf("Router.ExecuteGlobalAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}
