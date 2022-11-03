package engine

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// ExecuteAPI xxx
func (e *Engine) ExecuteAPIFromMap(api string, apiargs map[string]string) (result string, err error) {

	result = "" // pre-populate the most common result, when no err

	switch api {

	case "start", "stop":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("ExecuteAPI: missing process argument")
		}
		if api == "start" {
			err = e.ProcessManager.StartRunning(process)
		} else {
			err = e.ProcessManager.StopRunning(process)
		}
		return "", err

	case "nextalive":
		// acts like a timer, but it could wait for
		// some event if necessary
		time.Sleep(2 * time.Second)
		js := JsonObject(
			"event", "alive",
			"seconds", fmt.Sprintf("%f", e.Scheduler.aliveSecs),
			"attractmode", fmt.Sprintf("%v", e.Scheduler.attractModeIsOn),
		)
		return js, nil

	case "activate":
		go resolumeActivate()
		go biduleActivate()
		return "", nil

	case "sendlogs":
		return "", SendLogs()

	default:
		words := strings.Split(api, ".")
		// Any other single-word APIs (like get, set, load, etc)
		// are region-specific
		if len(words) <= 1 {
			region, regionok := apiargs["region"]
			if !regionok {
				region = "*"
			}
			result, err = e.Router.executeRegionAPI(region, api, apiargs)
			return result, err
		}
		// Here we handle APIs of the form {apitype}.{apisuffix}
		apitype := words[0]
		apisuffix := words[1]
		switch apitype {
		case "global":
			return e.executeGlobalAPI(apisuffix, apiargs)
		case "preset":
			return e.executePresetAPI(apisuffix, apiargs)
		case "sound":
			return e.executeSoundAPI(apisuffix, apiargs)
		default:
			return "", fmt.Errorf("ExecuteAPI: unknown prefix on api=%s", api)
		}
	}
	// unreachable
}

// handleRawJsonApi takes raw JSON (as a string of the form "{...}"") as an API and returns raw JSON
func (e *Engine) ExecuteAPIFromJson(rawjson string) (string, error) {
	args, err := StringMap(rawjson)
	if err != nil {
		return "", fmt.Errorf("Router.ExecuteAPIAsJson: bad format of JSON")
	}
	api, ok := args["api"]
	if !ok {
		return "", fmt.Errorf("Router.ExecuteAPIAsJson: no api value")
	}
	return e.ExecuteAPIFromMap(api, args)
}

func (e *Engine) executeGlobalAPI(api string, apiargs map[string]string) (result string, err error) {

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

		/*
					case "set_transpose":
						v, err := needFloatArg("value", api, apiargs)
						if err == nil {
							for _, motor := range e.Router.motors {
								motor.TransposePitch = int(v)
							}
							// log.Printf("set_transpose API set to %d\n", int(v))
						}

			case "set_transposeauto":
				b, err := needBoolArg("onoff", api, apiargs)
				if err == nil {
					e.Scheduler.transposeAuto = b
					// log.Printf("GlobalAPI: set_transposeauto b=%v\n", b)
					// Quantizing CurrentClick() to a beat or measure might be nice
					e.Scheduler.transposeNext = CurrentClick() + e.Scheduler.transposeClicks*oneBeat
					for _, motor := range e.Router.motors {
						motor.TransposePitch = 0
					}
				}

			case "set_scale":
				v, err := needStringArg("value", api, apiargs)
				if err == nil {
					for _, motor := range e.Router.motors {
						err = motor.SetOneParamValue("misc.scale", v)
						if err != nil {
							break
						}
					}
				}
		*/

	case "audio_reset":
		go e.Router.audioReset()

		/*
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
		*/

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		err = fmt.Errorf("Router.ExecuteGlobalAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func (e *Engine) executePresetAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		return PresetList(apiargs)

	case "load":
		presetName, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		preset := GetPreset(presetName)
		region, okregion := apiargs["region"]
		if !okregion {
			region = "*"
		}
		if preset.category == "quad" {
			err = preset.loadQuadPreset(region)
			if err != nil {
				log.Printf("loadQuad: preset=%s, err=%s", presetName, err)
				return "", err
			}
			e.Router.saveCurrentSnaps(region)
		} else {
			// It's a region preset
			motor, ok := e.Router.motors[region]
			if !ok {
				return "", fmt.Errorf("ExecutePresetAPI, no region named %s", region)
			}
			err = motor.applyPreset(preset)
			if err != nil {
				log.Printf("applyPreset: preset=%s, err=%s", presetName, err)
			} else {
				err = motor.saveCurrentSnap()
				if err != nil {
					log.Printf("saveCurrentSnap: err=%s", err)
				}
			}
		}
		return "", err

	case "save":
		presetName, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		region, okregion := apiargs["region"]
		if !okregion {
			return "", fmt.Errorf("missing region parameter")
		}
		motor, ok := e.Router.motors[region]
		if !ok {
			return "", fmt.Errorf("no motor named %s", region)
		}
		if strings.HasPrefix(presetName, "quad.") {
			return "", e.Router.saveQuadPreset(presetName)
		} else {
			return "", motor.saveCurrentAsPreset(presetName)
		}

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}
}

func (e *Engine) executeSoundAPI(api string, apiargs map[string]string) (result string, err error) {

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
