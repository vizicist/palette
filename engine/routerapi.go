package engine

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExecuteAPI xxx
func (r *Router) ExecuteAPI(api string, rawargs string) (result string, err error) {

	argsmap, e := StringMap(rawargs)
	if e != nil {
		result = ErrorResponse(fmt.Errorf("Router.ExecuteAPI: Unable to interpret value - %s", rawargs))
		log.Printf("Router.ExecuteAPI: bad rawargs value = %s\n", rawargs)
		return result, e
	}
	return r.ExecuteAPIAsMap(api, argsmap)
}

// ExecuteAPI xxx
func (r *Router) ExecuteAPIAsMap(api string, apiargs map[string]string) (result string, err error) {

	result = "" // pre-populate most common result

	switch api {

	case "start", "stop":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("ExecuteAPI: missing process argument")
		}
		if api == "start" {
			err = r.StartRunning(process)
		} else {
			err = r.StopRunning(process)
		}
		return "", err

	case "activate":
		go resolumeActivate()
		go biduleActivate()
		return "", nil

	case "sendlogs":
		return "", SendLogs()

	default:
		words := strings.Split(api, ".")
		// Singlw-word APIs (like get, set) are region-specific
		if len(words) <= 1 {
			region, regionok := apiargs["region"]
			if !regionok {
				region = "*"
			}
			result, err = r.executeRegionAPI(region, api, apiargs)
			return result, err
		}
		// Other APIs are of the form {apitype}.{api}
		apitype := words[0]
		apisuffix := ""
		if len(words) > 1 {
			apisuffix = words[1]
		}
		// So far there's only global.* APIs.
		if apitype == "global" {
			return r.executeGlobalAPI(apisuffix, apiargs)
		} else if apitype == "preset" {
			return r.executePresetAPI(apisuffix, apiargs)
		} else if apitype == "sound" {
			return r.executeSoundAPI(apisuffix, apiargs)
		} else {
			return "", fmt.Errorf("ExecuteAPI: unknown prefix on api=%s", api)
		}

	}
}

func presetArray(wantCategory string) ([]string, error) {

	result := make([]string, 0)

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
			result = append(result, thisCategory+"."+thisPreset)
		}
		return nil
	}

	presetsDir1 := filepath.Join(PaletteDataPath(), PresetsDir())
	err := filepath.Walk(presetsDir1, walker)
	if err != nil {
		log.Printf("filepath.Walk: err=%s\n", err)
		return nil, err
	}
	return result, nil
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

	presetsDir1 := filepath.Join(PaletteDataPath(), PresetsDir())
	err := filepath.Walk(presetsDir1, walker)
	if err != nil {
		log.Printf("filepath.Walk: err=%s\n", err)
	}
	result += "]"
	return result, nil
}

func (r *Router) executeRegionAPI(region string, api string, argsmap map[string]string) (result string, err error) {

	// XXX - Eventually, this should allow the region value to be "*" or multi-region

	switch api {

	case "event":
		return "", r.HandleInputEvent(argsmap)

	case "set":
		name, ok := argsmap["name"]
		if !ok {
			return "", fmt.Errorf("executeRegionAPI: missing name argument")
		}
		value, ok := argsmap["value"]
		if !ok {
			return "", fmt.Errorf("executeRegionAPI: missing value argument")
		}
		// Set value first
		for thisRegion, motor := range r.motors {
			if region == "*" || region == thisRegion {
				err = motor.SetOneParamValue(name, value)
				if err != nil {
					log.Printf("executeRegionAPI: set of %s failed, err=%s\n", name, err)
					// But don't fail completely, this might be for
					// parameters that no longer exist, and a hard failure may
					// cause more problems.
				}
			}
		}
		// then save it
		return "", r.saveCurrentSnaps(region)

	case "setparams":
		for name, value := range argsmap {
			if name == "region" {
				continue
			}
			for thisRegion, motor := range r.motors {
				if region == "*" || region == thisRegion {
					err = motor.SetOneParamValue(name, value)
					if err != nil {
						log.Printf("executeRegionAPI: set of %s failed, err=%s\n", name, err)
						// But don't fail completely, this might be for
						// parameters that no longer exist, and a hard failure may
						// cause more problems.
					}
				}
			}
		}
		return "", nil

	case "get":
		name, ok := argsmap["name"]
		if !ok {
			return "", fmt.Errorf("executeRegionAPI: missing name argument")
		}
		if region == "*" {
			return "", fmt.Errorf("executeRegionAPI: get can't handle *")
		}
		motor, ok := r.motors[region]
		if !ok {
			return "", fmt.Errorf("ExecuteRegionAPI: no region named %s", region)
		}
		return motor.params.paramValueAsString(name)

	default:
		// The region-specific APIs above are handled
		// here in the Router context, but for everything else,
		// we punt down to the region's motor.
		// region can be A, B, C, D, or *
		for tmpRegion, motor := range r.motors {
			if region == "*" || tmpRegion == region {
				_, err := motor.ExecuteAPI(api, argsmap, "")
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
	log.Printf("saveQuadPreset preset=%s\n", preset)
	for _, motor := range r.motors {
		log.Printf("starting motor=%s\n", motor.padName)
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
	return os.WriteFile(path, data, 0644)
}

func OldParameterName(nm string) bool {
	return nm == "sound.controller" || nm == "sound.controllerchan"
}

func (r *Router) loadQuadPresetRand() {

	arr, err := presetArray("quad")
	if err != nil {
		log.Printf("loadQuadPresetRand: err=%s\n", err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	log.Printf("loadQuadPresetRand: preset=%s", arr[rn])
	r.loadQuadPreset(arr[rn], "*")
}

func (r *Router) loadQuadPreset(preset string, applyToRegion string) error {

	path := ReadablePresetFilePath(preset)
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		return err
	}

	log.Printf("loadQuadPreset: preset=%s\n", preset)

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
		regionOfParam := words[0]
		motor, ok := r.motors[regionOfParam]
		if !ok {
			return fmt.Errorf("no region named %s", regionOfParam)
		}
		if applyToRegion != "*" && applyToRegion != regionOfParam {
			continue
		}
		// use words[1] so the motor doesn't see the region name
		parameterName := words[1]
		// We expect the parameter to be of the form
		// {category}.{parameter}, but old "quad" files
		// didn't include the category.
		if !strings.Contains(parameterName, ".") {
			log.Printf("loadQuadPreset: preset=%s parameter=%s is in OLD format, not supported", preset, parameterName)
			return fmt.Errorf("")
		}
		err = motor.SetOneParamValue(parameterName, value)
		if err != nil {
			if !OldParameterName(parameterName) {
				log.Printf("loadQuadPreset: name=%s err=%s\n", parameterName, err)
			}
			// Don't fail completely on individual failures,
			// some might be for parameters that no longer exist.
		}
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// preset, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing preset files.
	// This is similar to code in Motor.loadPreset, except we
	// have to do it for all for pads
	for _, c := range oneRouter.regionLetters {
		padName := string(c)
		motor := r.motors[padName]
		for nm, def := range ParamDefs {
			paramName := string(padName) + "-" + nm
			_, found := paramsmap[paramName]
			if !found {
				init := def.Init
				err = motor.SetOneParamValue(nm, init)
				if err != nil {
					// a hack to eliminate errors on a parameter that
					// still exists in some presets.
					if !OldParameterName(nm) {
						log.Printf("loadQuadPreset: %s, param=%s, init=%s, err=%s\n", preset, nm, init, err)
					}
					// Don't fail completely on individual failures,
					// some might be for parameters that no longer exist.
				}
			}
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
			// log.Printf("set_transpose API set to %d\n", int(v))
		}

	case "set_transposeauto":
		b, err := needBoolArg("onoff", api, apiargs)
		if err == nil {
			r.transposeAuto = b
			// log.Printf("GlobalAPI: set_transposeauto b=%v\n", b)
			// Quantizing CurrentClick() to a beat or measure might be nice
			r.transposeNext = CurrentClick() + r.transposeClicks*oneBeat
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

func (r *Router) saveCurrentSnaps(region string) error {
	// log.Printf("saveCurrentSnaps region=%s\n", region)
	if region == "*" {
		for _, motor := range r.motors {
			err := motor.saveCurrentSnap()
			if err != nil {
				return err
			}
		}
	} else {
		motor, ok := r.motors[region]
		if !ok {
			return fmt.Errorf("saveCurrentSnaps: no region named %s", region)
		}
		return motor.saveCurrentSnap()

	}
	return nil
}

func (r *Router) executePresetAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		return presetList(apiargs)

	case "load":
		preset, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		prefix, _ := PresetNameSplit(preset)
		region, okregion := apiargs["region"]
		if !okregion {
			region = "*"
		}
		if prefix == "quad" {
			err = r.loadQuadPreset(preset, region)
			if err != nil {
				log.Printf("loadQuad: preset=%s, err=%s", preset, err)
				return "", err
			}
			r.saveCurrentSnaps(region)
		} else {
			// It's a region preset
			motor, ok := r.motors[region]
			if !ok {
				return "", fmt.Errorf("ExecutePresetAPI, no region named %s", region)
			}
			err = motor.loadPreset(preset)
			if err != nil {
				log.Printf("loadPreset: preset=%s, err=%s", preset, err)
			} else {
				err = motor.saveCurrentSnap()
				if err != nil {
					log.Printf("saveCurrentSnap: err=%s", err)
				}
			}
		}
		return "", err

	case "save":
		preset, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		region, okregion := apiargs["region"]
		if !okregion {
			return "", fmt.Errorf("missing region parameter")
		}
		motor, ok := r.motors[region]
		if !ok {
			return "", fmt.Errorf("no motor named %s", region)
		}
		if strings.HasPrefix(preset, "quad.") {
			return "", r.saveQuadPreset(preset)
		} else {
			return "", motor.saveCurrentAsPreset(preset)
		}

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}
}

func (r *Router) executeSoundAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "playnote":
		notestr, oknote := apiargs["note"]
		if !oknote {
			return "", fmt.Errorf("missing note parameter")
		}
		_ = notestr
		log.Printf("sound.playnote API should be playing note=%s\n", notestr)
		return "", nil

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		err = fmt.Errorf("Router.ExecuteSoundAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}
