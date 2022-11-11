package engine

import (
	"fmt"
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
							for _, player := range e.Router.players {
								player.TransposePitch = int(v)
							}
							// Log.Debugf("set_transpose API set to %d\n", int(v))
						}

			case "set_transposeauto":
				b, err := needBoolArg("onoff", api, apiargs)
				if err == nil {
					e.Scheduler.transposeAuto = b
					// Log.Debugf("GlobalAPI: set_transposeauto b=%v\n", b)
					// Quantizing CurrentClick() to a beat or measure might be nice
					e.Scheduler.transposeNext = CurrentClick() + e.Scheduler.transposeClicks*oneBeat
					for _, player := range e.Router.players {
						player.TransposePitch = 0
					}
				}

			case "set_scale":
				v, err := needStringArg("value", api, apiargs)
				if err == nil {
					for _, player := range e.Router.players {
						err = player.SetOneParamValue("misc.scale", v)
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
					Log.Debugf("Hey, recordingFile wasn't nil?\n")
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
		Log.Debugf("Router.ExecuteAPI api=%s is not recognized\n", api)
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
		playerName, okplayer := apiargs["player"]
		if !okplayer {
			playerName = "*"
		}
		if preset.category == "quad" {
			// The playerName might be only a single player, and loadQuadPreset
			// will only load that one player from the quad preset
			err = preset.loadQuadPreset(playerName)
			if err != nil {
				Log.Debugf("loadQuad: preset=%s, err=%s", presetName, err)
				return "", err
			}
			e.Router.saveCurrentSnaps(playerName)
		} else {
			// It's a non-quad preset for a single player.
			// However, the playerName can still be "*" to apply to all players.
			err = preset.applyPreset(playerName)
			if err != nil {
				Log.Debugf("applyPreset: preset=%s, err=%s", presetName, err)
			} else {
				e.Router.saveCurrentSnaps(playerName)
			}
		}
		return "", err

	case "save":
		presetName, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		playerName, okplayer := apiargs["player"]
		if !okplayer {
			return "", fmt.Errorf("missing player parameter")
		}
		player, ok := e.Router.players[playerName]
		if !ok {
			return "", fmt.Errorf("no player named %s", playerName)
		}
		if strings.HasPrefix(presetName, "quad.") {
			return "", e.Router.saveQuadPreset(presetName)
		} else {
			return "", player.saveCurrentAsPreset(presetName)
		}

	default:
		Log.Debugf("Router.ExecuteAPI api=%s is not recognized\n", api)
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
		Log.Debugf("sound.playnote API should be playing note=%s\n", notestr)
		return "", nil

	default:
		Log.Debugf("Router.ExecuteAPI api=%s is not recognized\n", api)
		err = fmt.Errorf("Router.ExecuteSoundAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}
