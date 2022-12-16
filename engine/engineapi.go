package engine

import (
	"fmt"
	"strings"
)

// ExecuteAPI xxx
func (e *Engine) ExecuteAPIFromMap(api string, apiargs map[string]string) (result string, err error) {

	result = "" // pre-populate the most common result, when no err

	switch api {

	case "exit":
		e.StopMe()
		// notreached
		return "", nil

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

	case "activate":
		pproTask, err := TheTaskManager().GetTask("ppro")
		if err != nil {
			return "", err
		}
		return pproTask.methods.Api(pproTask, "activate", nil)

	case "sendlogs":
		return "", SendLogs()

	default:
		words := strings.Split(api, ".")
		// Single-word APIs (like get, set, load, etc) are layer APIs
		if len(words) <= 1 {
			return "", fmt.Errorf("single-word APIs no longer work")
			// return e.Router.executeLayerAPI(api, apiargs)
		}
		// Here we handle APIs of the form {apitype}.{apisuffix}
		apitype := words[0]
		apisuffix := words[1]
		switch apitype {
		case "engine":
			return e.executeEngineAPI(apisuffix, apiargs)
		case "preset":
			return e.executePresetAPI(apisuffix, apiargs)
		case "task":
			return e.executeTaskAPI(apisuffix, apiargs)
		case "sound":
			return e.executeSoundAPI(apisuffix, apiargs)
		case "layer":
			layerName := ExtractAndRemoveValue("layer", apiargs)
			if layerName == "" {
				return "", fmt.Errorf("no layer value")
			}
			layer := GetLayer(layerName)
			if layer == nil {
				return "", fmt.Errorf("no such layer: %s", layerName)
			}
			return layer.Api(apisuffix, apiargs)

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

func (e *Engine) executeEngineAPI(api string, apiargs map[string]string) (result string, err error) {

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
				SetLogTypeEnabled(s, b)
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
							ApplyToAllLayers(func(layer *Layer) {
								layer.TransposePitch = int(v)
							})
						}

			case "set_transposeauto":
				b, err := needBoolArg("onoff", api, apiargs)
				if err == nil {
					e.Scheduler.transposeAuto = b
					// Quantizing CurrentClick() to a beat or measure might be nice
					e.Scheduler.transposeNext = CurrentClick() + e.Scheduler.transposeClicks*OneBeat
					ApplyToAllLayers(func(layer *Layer) {
						layer.TransposePitch = 0
					})
				}

			case "set_scale":
				v, err := needStringArg("value", api, apiargs)
				if err == nil {
					ApplyToAllLayers(func(layer *Layer) {
						err = layer.SetOneParamValue("misc.scale", v)
						if err != nil {
							LogError(err)
						}
					})
				}
		*/

	case "audio_reset":
		go e.bidule.Reset()

		/*
			case "recordingStart":
				r.recordingOn = true
				if r.recordingFile != nil {
					r.recordingFile.Close()
				}
				r.recordingFile, err = os.Create(recordingsFile("LastRecording.json"))
				if err != nil {
					return "", err
				}
				r.recordingBegun = time.Now()
				if r.recordingOn {
					r.recordEvent("engine", "*", "start", "{}")
				}
			case "recordingSave":
				var name string
				name, err = needStringArg("name", api, apiargs)
				if err == nil {
					err = r.recordingSave(name)
				}

			case "recordingStop":
				if r.recordingOn {
					r.recordEvent("engine", "*", "stop", "{}")
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
		LogWarn("Router.ExecuteAPI api is not recognized\n", "api", api)
		err = fmt.Errorf("ExecuteEngineAPI: unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func (e *Engine) executePresetAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		return PresetList(apiargs)
	default:
		LogWarn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}
}

func (e *Engine) executeTaskAPI(api string, apiargs map[string]string) (result string, err error) {

	taskName, okTask := apiargs["task"]
	if !okTask {
		return "", fmt.Errorf("missing task parameter")
	}
	task, err := TheTaskManager().GetTask(taskName)
	if err != nil {
		return "", fmt.Errorf("task error, err=%s", err)
	}

	return task.methods.Api(task, api, apiargs)

}

func (e *Engine) executeSoundAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "playnote":
		notestr, oknote := apiargs["note"]
		if !oknote {
			return "", fmt.Errorf("missing note parameter")
		}
		LogInfo("sound.playnote API should be playing", "note", notestr)
		return "", nil

	default:
		LogWarn("Router.ExecuteAPI api is not recognized\n", "api", api)
		err = fmt.Errorf("Router.ExecuteSoundAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}
