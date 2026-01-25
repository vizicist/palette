package kit

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// NoProcess when true, skip auto-starting processes
var NoProcess bool

// ExecuteApi xxx
func ExecuteApi(api string, apiargs map[string]string) (result string, err error) {

	if api != "global.status" { // global.status happens every few seconds
		LogOfType("api", "ExecuteApi", "api", api, "apiargs", apiargs)
	}

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
	case "global":
		return ExecuteGlobalApi(apisuffix, apiargs)
	case "saved":
		return ExecuteSavedApi(apisuffix, apiargs)
	case "quad":
		if TheQuad != nil {
			return TheQuad.Api(apisuffix, apiargs)
		}
		return "", fmt.Errorf("no quad")
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

// ExecuteApiFromJson takes raw JSON (as a string of the form "{...}"") as an API and returns raw JSON
func ExecuteApiFromJson(rawjson string) (string, error) {

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in ExecuteApiFromJson called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in ExecuteApiFromJson has been called")
			LogError(err, "r", r, "stack", stacktrace)
		}
	}()

	args, err := StringMap(rawjson)
	if err != nil {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: bad format of JSON")
	}
	api := ExtractAndRemoveValueOf("api", args)
	if api == "" {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: no api value")
	}
	return ExecuteApi(api, args)
}

func ExecuteGlobalApi(api string, apiargs map[string]string) (result string, err error) {

	LogInfo("ExecuteGlobalApi", "api", api, "apiargs", apiargs)

	switch api {

	case "debugnil":
		// Generate a nil pointer panic
		var a *Engine
		a.SayDone()

	case "debugsched":
		return TheScheduler.ToString(), nil

	case "debugpending":
		return TheScheduler.PendingToString(), nil

	case "status":
		uptime := fmt.Sprintf("%f", Uptime())
		attractmode := strconv.FormatBool(TheAttractManager.AttractModeIsOn())
		natsConnected := strconv.FormatBool(natsIsConnected)
		if TheQuad == nil {
			result = JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
			)
		} else {
			result = JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
				"natsconnected", natsConnected,
				"A", Patchs["A"].Status(),
				"B", Patchs["B"].Status(),
				"C", Patchs["C"].Status(),
				"D", Patchs["D"].Status(),
			)
		}
		return result, nil

	case "attract":
		v, ok := apiargs["onoff"]
		if !ok {
			return "", fmt.Errorf("ExecuteGlobalApi: missing onoff parameter")
		}
		TheAttractManager.SetAttractMode(IsTrueValue(v))
		return "", nil

	case "load":
		fname, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("ExecuteGlobalApi: missing filename parameter")
		}
		err := LoadGlobalParamsFrom(fname, true)
		return "", err

	case "getboot", "getbootwithprefix":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("ExecuteGlobalApi: missing name parameter")
		}
		params, err := LoadParamValuesOfCategory("global", "_Boot")
		if err != nil {
			return "", err
		}
		if api == "getboot" {
			return params.Get(name)
		} else {
			return params.GetWithPrefix(name)
		}

	case "setboot":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", err
		}
		params, err := LoadParamValuesOfCategory("global", "_Boot")
		if err != nil {
			return "", err
		}
		err = params.SetParamWithString(name, value)
		if err != nil {
			return "", err
		}
		err = params.Save("global", "_Boot")
		if err != nil {
			return "", err
		}
		// Refresh EVERYTHING from _Boot, as if rebooting
		return "", LoadGlobalParamsFrom("_Boot", true)

	case "setbootfromcurrent":
		err = GlobalParams.Save("global", "_Boot")
		if err != nil {
			return "", err
		}
		// Refresh EVERYTHING from _Boot, as if rebooting
		return "", LoadGlobalParamsFrom("_Boot", true)

	case "set":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", err
		}
		err = SetAndApplyGlobalParam(name, value)
		if err != nil {
			return "", err
		}
		return "", SaveCurrentGlobalParams()

	case "setparams":
		for name, value := range apiargs {
			tmperr := SetAndApplyGlobalParam(name, value)
			if tmperr != nil {
				LogError(tmperr)
				err = tmperr
			}
		}
		if err != nil {
			return "", err
		}
		return "", SaveCurrentGlobalParams()

	case "get", "getwithprefix":

		LogInfo("ExecuteGlobalApi get 222", "api", api, "apiargs", apiargs)

		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("ExecuteGlobalApi: missing name parameter")
		}
		if api == "getwithprefix" {
			LogInfo("ExecuteGlobalApi 111", "api", api, "name", name)
			return GlobalParams.GetWithPrefix(name)
		} else {
			LogInfo("ExecuteGlobalApi 000", "api", api, "name", name)
			return GlobalParams.Get(name)
		}

	case "showclip":
		s, ok := apiargs["clipnum"]
		if !ok {
			return "", fmt.Errorf("global.showimage: missing filename parameter")
		}
		clipNum, err := strconv.Atoi(s)
		if err != nil {
			return "", fmt.Errorf("global.showimage: bad clipnum parameter")
		}
		TheResolume().showClip(clipNum)
		return "", nil

	case "startrecording":
		return TheEngine.StartRecording()

	case "stoprecording":
		return TheEngine.StopRecording()

	case "startplayback":
		fname, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("ExecuteGlobalApi: missing filename parameter")
		}
		return "", TheEngine.StartPlayback(fname)

	case "save":
		filename, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("ExecuteGlobalApi: missing filename parameter")
		}
		return "", GlobalParams.Save("global", filename)

	case "done":
		TheEngine.SayDone() // needed for clean exit when profiling
		return "", nil

	case "audio_reset":
		go TheBidule().Reset()

	case "sendlogs":
		return "", ArchiveLogs()

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
		tag, ok := apiargs["tag"]
		if !ok {
			tag = "A"
		}
		var pos CursorPos
		xs := apiargs["x"]
		ys := apiargs["y"]
		zs := apiargs["z"]
		if xs == "" || ys == "" || zs == "" {
			return "", fmt.Errorf("playcursor: missing x, y, or z value")
		}
		var x, y, z float64
		if !GetFloat(xs, &x) || !GetFloat(ys, &y) || !GetFloat(zs, &z) {
			return "", fmt.Errorf("playcursor: bad x,y,z value")
		}
		pos = CursorPos{x, y, z}
		go TheCursorManager.PlayCursor(tag, dur, pos)
		return "", nil

	default:
		LogWarn("Router.ExecuteApi api is not recognized\n", "api", api)
		err = fmt.Errorf("ExecuteEngineApi: unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func GetFloat(value string, f *float64) bool {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = v
		return true
	}
}

/*
func GetFloat32(value string, f *float32) bool {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = float32(v)
		return true
	}
}
*/

func GetInt(value string, i *int64) bool {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*i = v
		return true
	}
}

func ActivateGlobalParam(name string) {
	val, err := GetParam(name)
	LogIfError(err)
	err = SetAndApplyGlobalParam(name, val)
	LogIfError(err)

}

func SetAndApplyGlobalParam(name string, value string) (err error) {
	err = GlobalParams.SetParamWithString(name, value)
	if err != nil {
		return err
	}
	return ApplyGlobalParam(name, value)
}

func ApplyGlobalParam(name string, value string) (err error) {

	_, ok := ParamDefs[name]
	if !ok {
		err := fmt.Errorf("ApplyParam: unknown parameter %s", name)
		LogError(err)
		return err
	}

	var f float64
	var i int64

	// LogOfType("params", "Engine.ApplyParam", "name", name, "value", value)

	if strings.HasPrefix(name, "global.process.") {
		if NoProcess {
			return nil
		}
		process := strings.TrimPrefix(name, "global.process.")
		if TheProcessManager.IsAvailable(process) {
			if IsTrueValue(value) {
				err = TheProcessManager.StartRunning(process)
			} else {
				err = TheProcessManager.StopRunning(process)
			}
			if err != nil {
				LogError(err)
				return err
			}
		}
	}

	switch name {

	case "global.attract":
		TheAttractManager.SetAttractMode(IsTrueValue(value))

	case "global.attractenabled":
		TheAttractManager.SetAttractEnabled(IsTrueValue(value))

	case "global.attractidlesecs":
		if GetInt(value, &i) {
			if i < 15 {
				LogWarn("global.attractidlesecs is too low, forcing to 15")
				i = 15
			}
			TheAttractManager.IdleSecs = float64(i)
		}
	case "global.attractgestureinterval":
		if GetFloat(value, &f) {
			TheAttractManager.GestureInterval = f
		}
	case "global.attractpresetchangeinterval":
		if GetFloat(value, &f) {
			TheAttractManager.PresetChangeInterval = f
		}
	case "global.attractgesturenumsteps":
		if GetFloat(value, &f) {
			TheAttractManager.GestureNumSteps = f
		}
	case "global.attractgestureduration":
		if GetFloat(value, &f) {
			TheAttractManager.GestureDuration = f
		}
	case "global.attractgestureminlength":
		if GetFloat(value, &f) {
			TheAttractManager.GestureMinLength = f
		}
	case "global.attractgesturemaxlength":
		if GetFloat(value, &f) {
			TheAttractManager.GestureMaxLength = f
		}
	case "global.attractgestureminz":
		if GetFloat(value, &f) {
			TheAttractManager.GestureZMin = f
		}
	case "global.attractgesturemaxz":
		if GetFloat(value, &f) {
			TheAttractManager.GestureZMax = f
		}
	case "global.erae":
		b, _ := GetParamBool("global.erae")
		if b {
			TheErae.EraeApiModeEnable()
			TheMidiIO.SetMidiInput("Erae 2")
		}

	case "global.looping_fadethreshold":
		if GetFloat(value, &f) {
			TheCursorManager.LoopThreshold = f
		}

	case "global.looping_override", "global.looping_fade", "global.looping_beats":
		err := GlobalParams.SetParamWithString(name, value)
		if err != nil {
			LogError(err)
			return err
		}

	case "global.midithru":
		TheRouter.midithru = IsTrueValue(value)

	case "global.midisetexternalscale":
		TheRouter.midisetexternalscale = IsTrueValue(value)

	case "global.midithruscadjust":
		TheRouter.midiThruScadjust = IsTrueValue(value)

	case "global.oscoutput":
		TheEngine.oscoutput = IsTrueValue(value)

	case "global.autotranspose":
		TheEngine.autoTransposeOn = IsTrueValue(value)

	case "global.autotransposebeats":
		if GetInt(value, &i) {
			TheEngine.SetAutoTransposeBeats(int(i))
		}
	case "global.transpose":
		if GetInt(value, &i) {
			TheEngine.SetTranspose(int(i))
		}
	case "global.log":
		SetLogTypes(value)

	case "global.midiinput":
		TheMidiIO.SetMidiInput(value)

	case "global.processchecksecs":
		if GetFloat(value, &f) {
			TheProcessManager.processCheckSecs = f
		}

	case "global.obsstream":
		if IsTrueValue(value) {
			err := ObsCommand("streamstart")
			LogIfError(err)
		} else {
			// Ignore errors, since it may not be running at all
			_ = ObsCommand("streamstop")
		}
	}
	return nil
}

// func (e *Engine) SetParam(name string, value string) (err error) {
// 	LogOfType("params", "Engine.SetParam", "name", name, "value", value)
// 	return GlobalParams.SetParamWithString(name, value)
// }

func ExecuteSavedApi(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		category := optionalStringArg("category", apiargs, "*")
		return SavedListAsString(category)
	default:
		LogWarn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecuteSavedApi unrecognized api=%s", api)
	}
}
