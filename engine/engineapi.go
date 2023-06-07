package engine

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// ExecuteAPI xxx
func (e *Engine) ExecuteAPI(api string, apiargs map[string]string) (result string, err error) {

	LogOfType("api", "ExecuteAPI", "api", api, "apiargs", apiargs)

	words := strings.Split(api, ".")
	if len(words) <= 1 {
		err = fmt.Errorf("single-word APIs no longer work, api=%s", api)
		LogIfError(err)
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
	case "quadpro":
		if TheQuadPro != nil {
			return TheQuadPro.Api(apisuffix, apiargs)
		}
		return "", fmt.Errorf("no quadpro")
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
		return "", fmt.Errorf("unknown apitype: %s", apitype)
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

	case "status":
		result = JsonObject(
			"uptime", fmt.Sprintf("%f", Uptime()),
			"attractmode", fmt.Sprintf("%v", TheAttractManager.AttractModeIsOn()),
		)
		return result, nil

	case "attract":
		v, ok := apiargs["onoff"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing onoff parameter")
		}
		TheAttractManager.SetAttractMode(IsTrueValue(v))
		return "", nil

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
				LogIfError(e)
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
		return e.params.Get(name)

	case "startprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing process parameter")
		}
		return "", TheProcessManager.StartRunning(process)

	case "stopprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing process parameter")
		}
		err := TheProcessManager.StopRunning(process)
		return "", err

	case "startrecording":
		return e.StartRecording()

	case "stoprecording":
		return e.StopRecording()

	case "startplayback":
		fname, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing filename parameter")
		}
		return "", e.StartPlayback(fname)

	case "save":
		filename, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("executeEngineAPI: missing filename parameter")
		}
		return "", e.params.Save("engine", filename)

	case "exit":
		e.StopMe()
		go e.stopAfterDelay()
		return "", nil

	case "audio_reset":
		go LogIfError(TheBidule().Reset())

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

	case "playcursor":
		var dur = 500 * time.Millisecond // default
		s, ok := apiargs["dur"]
		if ok {
			tmp, err := time.ParseDuration(s)
			if err != nil {
				return "", err
			}
			dur = tmp
		}
		source, ok := apiargs["source"]
		if !ok {
			source = "A"
		}
		xs := apiargs["x"]
		ys := apiargs["y"]
		zs := apiargs["z"]
		var x, y, z float32
		if xs == "" {
			x = rand.Float32()
			y = rand.Float32()
			z = rand.Float32()
		} else {
			if !e.getFloat32(xs, &x) || !e.getFloat32(ys, &y) || !e.getFloat32(zs, &z) {
				return "", fmt.Errorf("playcursor: bad x,y,z value")
			}
		}
		go TheCursorManager.PlayCursor(source, dur, x, y, z)
		return "", nil

	default:
		LogWarn("Router.ExecuteAPI api is not recognized\n", "api", api)
		err = fmt.Errorf("ExecuteEngineAPI: unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func (e *Engine) getFloat(value string, f *float64) bool {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = v
		return true
	}
}

func (e *Engine) getFloat32(value string, f *float32) bool {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = float32(v)
		return true
	}
}

func (e *Engine) getInt(value string, i *int64) bool {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*i = v
		return true
	}
}

func (e *Engine) Set(name string, value string) error {
	var f float64
	var i int64
	switch name {

	case "engine.attract":
		TheAttractManager.SetAttractMode(IsTrueValue(value))

	case "engine.attractenabled":
		TheAttractManager.attractEnabled = IsTrueValue(value)

	case "engine.attractchecksecs":
		if e.getFloat(value, &f) {
			TheAttractManager.attractCheckSecs = f
		}
	case "engine.attractchangeinterval":
		if e.getFloat(value, &f) {
			TheAttractManager.attractChangeInterval = f
		}
	case "engine.attractgestureinterval":
		if e.getFloat(value, &f) {
			TheAttractManager.attractGestureInterval = f
		}
	case "engine.attractidlesecs":
		if e.getInt(value, &i) {
			if i < 10 {
				LogWarn("engine.attractidlesecs is too low, forcing to 10")
				i = 10
			}
			TheAttractManager.attractIdleSecs = float64(i)
		}
	case "engine.midithru":
		TheRouter.midithru = IsTrueValue(value)

	case "engine.midisetexternalscale":
		TheRouter.midisetexternalscale = IsTrueValue(value)

	case "engine.midithruscadjust":
		TheRouter.midiThruScadjust = IsTrueValue(value)

	case "engine.oscoutput":
		e.oscoutput = IsTrueValue(value)

	case "engine.autotranspose":
		e.autoTransposeOn = IsTrueValue(value)

	case "engine.autotransposebeats":
		if e.getInt(value, &i) {
			e.SetAutoTransposeBeats(int(i))
		}
	case "engine.transpose":
		if e.getInt(value, &i) {
			e.SetTranspose(int(i))
		}
	case "engine.log":
		e.ResetLogTypes(value)

	case "engine.midiinput":
		TheMidiIO.SetMidiInput(value)

	case "engine.processchecksecs":
		if e.getFloat(value, &f) {
			TheProcessManager.processCheckSecs = f
		}
	}

	LogOfType("params", "Engine.Set", "name", name, "value", value)
	return e.params.Set(name, value)
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
