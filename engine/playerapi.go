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
func (ctx *AgentContext) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	DebugLogOfType("api", "Agent.ExecutAPI called", "api", api, "args", args)
	// The caller can provide rawargs if it's already known, but if not provided, we create it
	if rawargs == "" {
		rawargs = MapString(args)
	}

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if strings.HasPrefix(api, "visual.") {
		msg := osc.NewMessage("/api")
		msg.Append(strings.TrimPrefix(api, "visual."))
		msg.Append(rawargs)
		ctx.toFreeFramePluginForLayer(msg)
	}

	switch api {

	case "send":
		ctx.sendAllParameters()
		return "", err

		//	case "loop_recording":
		//		v, e := needBoolArg("onoff", api, args)
		//		if e == nil {
		//			ctx.loopIsRecording = v
		//		} else {
		//			err = e
		//		}
		//
		//	case "loop_playing":
		//		v, e := needBoolArg("onoff", api, args)
		//		if e == nil && v != player.loopIsPlaying {
		//			player.loopIsPlaying = v
		//			TheEngine().Scheduler.SendAllPendingNoteoffs()
		//		} else {
		//			err = e
		//		}
		//
		//	case "loop_clear":
		//		// player.loop.Clear()
		//		// player.clearGraphics()
		//		player.sendANO()
		//
		//	case "loop_length":
		//		i, e := needIntArg("value", api, args)
		//		if e == nil {
		//			nclicks := Clicks(i)
		//			if nclicks != player.loopLength {
		//				player.loopLength = nclicks
		//				// player.loopSetLength(nclicks)
		//			}
		//		} else {
		//			err = e
		//		}
		//
		//	case "loop_fade":
		//		f, e := needFloatArg("fade", api, args)
		//		if e == nil {
		//			player.fadeLoop = f
		//		} else {
		//			err = e
		//		}
		//
		//	case "ANO":
		//		player.sendANO()

		/*
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
		*/

	default:
		err = fmt.Errorf("Player.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}

func (ctx *AgentContext) sendAllParameters() {
	for nm := range ctx.params.values {
		val, err := ctx.params.paramValueAsString(nm)
		if err != nil {
			LogError(err)
			// Don't fail completely
			continue
		}
		// This assumes that if you set a parameter to the same value,
		// that it will re-send the mesasges to Resolume for visual.* params
		err = ctx.SetOneParamValue(nm, val)
		if err != nil {
			LogError(err)
			// Don't fail completely
		}
	}
}

func (ctx *AgentContext) applyParamsMap(presetType string, paramsmap map[string]interface{}) error {

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
		err := ctx.SetOneParamValue(fullname, val)
		if err != nil {
			LogError(err)
			// Don't abort the whole load, i.e. we are tolerant
			// of unknown parameters or errors in the preset
		}
	}
	return nil
}

func (ctx *AgentContext) restoreCurrentSnap(layerName string) {
	preset, err := LoadPreset("snap._Current_" + layerName)
	if err != nil {
		LogError(err)
		return
	}
	err = preset.ApplyTo(ctx)
	if err != nil {
		LogError(err)
	}
}

func (ctx *AgentContext) saveCurrentAsPreset(presetName string) error {
	preset, err := LoadPreset(presetName)
	if err != nil {
		return err
	}
	path := preset.WriteableFilePath()
	return ctx.saveCurrentSnapInPath(path)
}

func (ctx *AgentContext) saveCurrentSnapInPath(path string) error {

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := ctx.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		valstring, e := ctx.params.paramValueAsString(fullName)
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
