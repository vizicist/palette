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
		go resolumeActivate()
		go biduleActivate()
		return "", nil

	case "sendlogs":
		return "", SendLogs()

	default:
		words := strings.Split(api, ".")
		// Single-word APIs (like get, set, load, etc) are player APIs
		if len(words) <= 1 {
			return e.Router.executePlayerAPI(api, apiargs)
		}
		// Here we handle APIs of the form {apitype}.{apisuffix}
		apitype := words[0]
		apisuffix := words[1]
		switch apitype {
		case "global":
			return e.executeGlobalAPI(apisuffix, apiargs)
		case "preset":
			return e.executePresetAPI(apisuffix, apiargs)
		case "task":
			return e.executeTaskAPI(apisuffix, apiargs)
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
							ApplyToAllPlayers(func(player *Player) {
								player.TransposePitch = int(v)
							})
						}

			case "set_transposeauto":
				b, err := needBoolArg("onoff", api, apiargs)
				if err == nil {
					e.Scheduler.transposeAuto = b
					// Quantizing CurrentClick() to a beat or measure might be nice
					e.Scheduler.transposeNext = CurrentClick() + e.Scheduler.transposeClicks*OneBeat
					ApplyToAllPlayers(func(player *Player) {
						player.TransposePitch = 0
					})
				}

			case "set_scale":
				v, err := needStringArg("value", api, apiargs)
				if err == nil {
					ApplyToAllPlayers(func(player *Player) {
						err = player.SetOneParamValue("misc.scale", v)
						if err != nil {
							LogError(err)
						}
					})
				}
		*/

	case "audio_reset":
		go e.Router.audioReset()

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
		Warn("Router.ExecuteAPI api is not recognized\n", "api", api)
		err = fmt.Errorf("Router.ExecuteGlobalAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func (e *Engine) executePresetAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		return PresetList(apiargs)
	default:
		Warn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}
}

type ApiEvent struct {
	api     string
	apiargs map[string]string
}

func (e *Engine) executeTaskAPI(api string, apiargs map[string]string) (result string, err error) {

	taskName, okTask := apiargs["task"]
	if !okTask {
		return "", fmt.Errorf("missing agent parameter")
	}
	task, err := TheRouter().taskManager.GetTask(taskName)

	return task.methods.Api(task, api, apiargs)

}

func (e *Engine) executeSoundAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "playnote":
		notestr, oknote := apiargs["note"]
		if !oknote {
			return "", fmt.Errorf("missing note parameter")
		}
		Info("sound.playnote API should be playing", "note", notestr)
		return "", nil

	default:
		Warn("Router.ExecuteAPI api is not recognized\n", "api", api)
		err = fmt.Errorf("Router.ExecuteSoundAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}
