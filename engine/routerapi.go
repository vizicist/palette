package engine

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
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

func (r *Router) saveQuadPreset(presetName string) error {

	preset := GetPreset(presetName)
	// wantCategory is sound, visual, effect, snap, or quad
	path := preset.WriteableFilePath()
	s := "{\n    \"params\": {\n"

	sep := ""
	log.Printf("saveQuadPreset preset=%s\n", presetName)
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
	preset := GetPreset(arr[rn])
	preset.loadQuadPreset("*")
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
