package engine

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ExecuteAPI xxx
func (e *Engine) ExecuteAPI(api string, apiargs map[string]string) (result string, err error) {

	LogOfType("api", "ExecuteAPI", "api", api, "apiargs", apiargs)

	result = "" // pre-populate the most common result, when no err

	switch api {

	default:
		words := strings.Split(api, ".")
		if len(words) <= 1 {
			err = fmt.Errorf("single-word APIs no longer work, api=%s", api)
			LogError(err)
			return "", err
		}
		// Here we handle APIs of the form {apitype}.{apisuffix}
		apitype := words[0]
		apisuffix := words[1]
		switch apitype {
		case "engine":
			return e.executeEngineAPI(apisuffix, apiargs)
		case "saved":
			return e.executeSavedAPI(apisuffix, apiargs)
		case "sound":
			return e.executeSoundAPI(apisuffix, apiargs)
		case "patch":
			patchName := ExtractAndRemoveValueOf("patch", apiargs)
			if patchName == "" {
				return "", fmt.Errorf("no patch value")
			}
			patch := GetPatch(patchName)
			if patch == nil {
				return "", fmt.Errorf("no such patch: %s", patchName)
			}
			return patch.Api(apisuffix, apiargs)
		default:
			// If it's not one of the reserved aent names, see if it's a registered one
			ctx, ok := Plugins[apitype]
			if !ok {
				return "", fmt.Errorf("No such plugin: %s", apitype)
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
	api := ExtractAndRemoveValueOf("api", args)
	if api == "" {
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
		_ = StopRunning("bidule")
		_ = StopRunning("resolume")
		_ = StopRunning("keykit")
		_ = StopRunning("gui")
		go e.stopAfterDelay()
		return "", nil

	case "status":
		result = JsonObject(
			"uptime", fmt.Sprintf("%f", Uptime()),
			"attractmode", fmt.Sprintf("%v", TheAttractManager.attractModeIsOn),
		)
		return result, nil

	case "set":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", err
		}
		err = e.Set(name, value)
		if err != nil {
			return "", err
		}
		return "", e.SaveCurrent()

	case "setparams":
		for name, value := range apiargs {
			e := e.Set(name, value)
			if e != nil {
				LogError(e)
				err = e
			}
		}
		if err != nil {
			return "", err
		}
		return "", e.SaveCurrent()

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing name parameter")
		}
		return e.Get(name), nil

	case "startprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing process parameter")
		}
		err := TheProcessManager.StartRunning(process)
		if err != nil {
			return "", err
		}
		err = TheProcessManager.Activate(process)
		return "", err

	case "stopprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing process parameter")
		}
		err := TheProcessManager.StopRunning(process)
		return "", err

	case "save":
		filename, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing filename parameter")
		}
		return "", e.params.Save("engine", filename)

	case "exit":
		e.StopMe()
		return "", nil

	case "activate":
		// Force Activate even if already activated
		return "", TheProcessManager.Activate("all")

	case "audio_reset":
		go LogError(TheBidule().Reset())

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
		return "", fmt.Errorf("debug API has been removed")

	case "set_tempo_factor":
		v, err := needFloatArg("value", api, apiargs)
		if err == nil {
			ChangeClicksPerSecond(float64(v))
		}

	case "clearexternalscale":
		ClearExternalScale()

	case "set_transpose":
		LogWarn("set_transpose API needs work")
		// v, err := needFloatArg("value", api, apiargs)
		// if err == nil {
		// ApplyToAllPatchs(func(patch *Patch) {
		// 	patch.TransposePitch = int(v)
		// })
		// }

	case "set_transposeauto":
		LogWarn("set_transposeauto API needs work")
		// b, err := needBoolArg("onoff", api, apiargs)
		// if err == nil {
		// 	e.Scheduler.transposeAuto = b
		// 	// Quantizing CurrentClick() to a beat or measure might be nice
		// 	e.Scheduler.transposeNext = CurrentClick() + e.Scheduler.transposeClicks*OneBeat
		// 	ApplyToAllPatchs(func(patch *Patch) {
		// 		patch.TransposePitch = 0
		// 	})
		// }

	case "set_scale":
		LogWarn("set_scale API needs work")
		/*
			v, err := needStringArg("value", api, apiargs)
			if err == nil {
				ApplyToAllPatchs(func(patch *Patch) {
					err = patch.SetOneParamValue("misc.scale", v)
					if err != nil {
						LogError(err)
					}
				})
			}
		*/

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

func (e *Engine) SaveCurrent() (err error) {
	return e.params.Save("engine", "_Current")
}

func (e *Engine) LoadCurrent() (err error) {
	path := SavedFilePath("engine", "_Current")
	paramsmap, err := LoadParamsMap(path)
	if err == nil {
		e.params.ApplyValuesFromMap("engine", paramsmap, e.Set)
	}
	return err
}

func (e *Engine) Set(name string, value string) error {
	// LogInfo("Engine.Set", "name", name, "value", value)
	switch name {
	case "engine.oscoutput":
		e.oscOutput = IsTrueValue(value)
	case "engine.debug":
		e.ResetLogTypes(value)
	case "engine.attract":
		TheAttractManager.setAttractMode(IsTrueValue(value))
	case "engine.attractidlesecs":
		var f float64
		f, err := strconv.ParseFloat(value, 32)
		if err != nil {
			LogError(err)
		} else {
			TheAttractManager.attractIdleSecs = f
		}
	}
	return e.params.Set(name, value)
}

func (e *Engine) Get(name string) string {
	return e.params.Get(name)
}

func (e *Engine) GetWithDefault(nm string, dflt string) string {
	if e.params.Exists(nm) {
		return e.params.Get(nm)
	} else {
		return dflt
	}
}

// ParamBool returns bool value of nm, or false if nm not set
func (e *Engine) ParamBool(nm string) bool {
	v := e.Get(nm)
	if v == "" {
		return false
	}
	return IsTrueValue(v)
}

func (e *Engine) EngineParamIntWithDefault(nm string, dflt int) int {
	s := e.Get(nm)
	if s == "" {
		return dflt
	}
	var val int
	nfound, err := fmt.Sscanf(s, "%d", &val)
	if nfound == 0 || err != nil {
		LogError(err)
		return dflt
	}
	return val
}

func (e *Engine) EngineParamFloatWithDefault(nm string, dflt float64) float64 {
	s := e.Get(nm)
	if s == "" {
		return dflt
	}
	var f float64
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		LogError(err)
		return dflt
	}
	return f
}

func (e *Engine) executeSavedAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		return SavedList(apiargs)
	default:
		LogWarn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecuteSavedAPI unrecognized api=%s", api)
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
