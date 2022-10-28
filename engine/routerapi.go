package engine

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
)

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

	arr, err := PresetArray("quad")
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
	for _, c := range r.regionLetters {
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
