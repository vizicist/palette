package engine

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ExecuteAPI xxx
func (e *Engine) ExecuteAPI(api string, apiargs map[string]string) (result string, err error) {

	DebugLogOfType("api", "ExecuteAPI", "api", api, "apiargs", apiargs)

	result = "" // pre-populate the most common result, when no err

	switch api {

	default:
		words := strings.Split(api, ".")
		if len(words) <= 1 {
			return "", fmt.Errorf("single-word APIs no longer work")
		}
		// Here we handle APIs of the form {apitype}.{apisuffix}
		apitype := words[0]
		apisuffix := words[1]
		switch apitype {
		case "engine":
			return e.executeEngineAPI(apisuffix, apiargs)
		case "preset":
			return e.executePresetAPI(apisuffix, apiargs)
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
			// If it's not one of the reserved aent names, see if it's a registered one
			ctx, err := e.PluginManager.GetPluginContext(apitype)
			if err != nil {
				return "", err
			}
			return ctx.api(ctx, apisuffix, apiargs)
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
	return e.ExecuteAPI(api, args)
}

func (e *Engine) executeEngineAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "stop":
		go e.stopAfterDelay()
		return "", nil

	case "stopall":
		CallApiOnAllPlugins("stop", map[string]string{})
		go e.stopAfterDelay()
		return "", nil

	case "exit":
		e.StopMe()
		return "", nil

	case "sendlogs":
		return "", SendLogs()

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

	// case "audio_reset":
	// 	go e.bidule.Reset()

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

func (e *Engine) stopAfterDelay() {
	time.Sleep(500 * time.Millisecond)
	LogInfo("Engine.stop: calling os.Exit(0)")
	os.Exit(0)
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
