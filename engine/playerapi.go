package engine

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type ParamsMap map[string]interface{}

// ExecuteAPI xxx
func (player *Player) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	DebugLogOfType("player", "PlayerAPI", "api", api, "args", args)
	// The caller can provide rawargs if it's already known, but if not provided, we create it
	if rawargs == "" {
		rawargs = MapString(args)
	}

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if strings.HasPrefix(api, "visual.") {
		msg := osc.NewMessage("/api")
		msg.Append(strings.TrimPrefix(api, "visual."))
		msg.Append(rawargs)
		player.toFreeFramePluginForLayer(msg)
	}

	switch api {

	case "send":
		DebugLogOfType("*", "API send should be sending all parameters", "player", player.padName)
		player.sendAllParameters()
		return "", err

		/*
			case "loop_recording":
				v, e := needBoolArg("onoff", api, args)
				if e == nil {
					player.loopIsRecording = v
				} else {
					err = e
				}

			case "loop_playing":
				v, e := needBoolArg("onoff", api, args)
				if e == nil && v != player.loopIsPlaying {
					player.loopIsPlaying = v
					player.terminateActiveNotes()
				} else {
					err = e
				}

			case "loop_clear":
				player.loop.Clear()
				player.clearGraphics()
				player.sendANO()

			case "loop_comb":
				player.loopComb()

			case "loop_length":
				i, e := needIntArg("value", api, args)
				if e == nil {
					nclicks := Clicks(i)
					if nclicks != player.loop.length {
						player.loop.SetLength(nclicks)
					}
				} else {
					err = e
				}

			case "loop_fade":
				f, e := needFloatArg("fade", api, args)
				if e == nil {
					player.fadeLoop = f
				} else {
					err = e
				}
		*/

	case "ANO":
		player.sendANO()

	case "midi_thru":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			player.MIDIThru = v
		} else {
			err = e
		}

	case "midi_setscale":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			player.MIDISetScale = v
		} else {
			err = e
		}

	case "midi_usescale":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			player.MIDIUseScale = v
		} else {
			err = e
		}

	case "clearexternalscale":
		player.clearExternalScale()
		player.MIDINumDown = 0

	case "midi_quantized":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			player.MIDIQuantized = v
		}

	case "midi_thruscadjust":
		v, e := needBoolArg("onoff", api, args)
		if e == nil {
			player.MIDIThruScadjust = v
		} else {
			err = e
		}

	case "set_transpose":
		v, e := needIntArg("value", api, args)
		if e == nil {
			player.TransposePitch = v
			Info("player API set_transpose", "pitch", v)
		} else {
			err = e
		}

	default:
		err = fmt.Errorf("Player.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}

func (player *Player) sendAllParameters() {
	for nm := range player.params.values {
		val, err := player.params.paramValueAsString(nm)
		if err != nil {
			LogError(err)
			// Don't fail completely
			continue
		}
		// This assumes that if you set a parameter to the same value,
		// that it will re-send the mesasges to Resolume for visual.* params
		err = player.SetOneParamValue(nm, val)
		if err != nil {
			LogError(err)
			// Don't fail completely
		}
	}
}

func (player *Player) applyParamsMap(presetType string, paramsmap map[string]interface{}) error {

	// Currently, no errors are ever returned, but log messages are generated.

	for name, ival := range paramsmap {
		val, okval := ival.(string)
		if !okval {
			Warn("value isn't a string in params json", "name", name, "value", val)
			continue
		}
		fullname := name
		thisCategory, _ := PresetNameSplit(fullname)
		// Only include ones that match the presetType
		if presetType != "snap" && thisCategory != presetType {
			continue
		}
		// This is where the parameter values get applied,
		// which may trigger things (like sending OSC)
		err := player.SetOneParamValue(fullname, val)
		if err != nil {
			LogError(err)
			// Don't abort the whole load, i.e. we are tolerant
			// of unknown parameters or errors in the preset
		}
	}
	return nil
}

func (player *Player) restoreCurrentSnap() {
	playerName := player.padName
	preset := GetPreset("snap._Current_" + playerName)
	err := preset.applyPreset(playerName)
	if err != nil {
		LogError(err)
	}
}

func (player *Player) saveCurrentAsPreset(presetName string) error {
	preset := GetPreset(presetName)
	path := preset.WriteableFilePath()
	return player.saveCurrentSnapInPath(path)
}

func (player *Player) saveCurrentSnap() error {
	return player.saveCurrentAsPreset("snap._Current_" + player.padName)
}

func (player *Player) saveCurrentSnapInPath(path string) error {

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := player.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		valstring, e := player.params.paramValueAsString(fullName)
		if e != nil {
			LogError(e)
			continue
		}
		s += fmt.Sprintf("%s        \"%s\":\"%s\"", sep, fullName, valstring)
		sep = ",\n"
	}
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}
