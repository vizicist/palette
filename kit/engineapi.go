package engine

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ExecuteApi xxx
func (e *Engine) ExecuteApi(api string, apiargs map[string]string) (result string, err error) {

	LogOfType("api", "ExecuteApi", "api", api, "apiargs", apiargs)

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
		return e.executeEngineApi(apisuffix, apiargs)
	case "saved":
		return e.executeSavedApi(apisuffix, apiargs)
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
func (e *Engine) ExecuteApiFromJson(rawjson string) (string, error) {
	args, err := StringMap(rawjson)
	if err != nil {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: bad format of JSON")
	}
	api := ExtractAndRemoveValueOf("api", args)
	if api == "" {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: no api value")
	}
	return e.ExecuteApi(api, args)
}

func (e *Engine) executeEngineApi(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "debugsched":
		return TheScheduler.ToString(), nil

	case "debugpending":
		return TheScheduler.PendingToString(), nil

	case "status":
		uptime := fmt.Sprintf("%f", Uptime())
		attractmode := fmt.Sprintf("%v", TheAttractManager.AttractModeIsOn())
		if TheQuadPro == nil {
			result = JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
			)
		} else {
			result = JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
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
			return "", fmt.Errorf("executeEngineApi: missing onoff parameter")
		}
		TheAttractManager.SetAttractMode(IsTrueValue(v))
		return "", nil

	case "set":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", err
		}
		err = GlobalParams.SetParamWithString(name, value)
		if err != nil {
			return "", err
		}
		err = ApplyParam(name, value)
		if err != nil {
			return "", err
		}
		return "", SaveCurrent()

	case "setparams":
		for name, value := range apiargs {
			tmperr := GlobalParams.SetParamWithString(name, value)
			if tmperr != nil {
				LogError(tmperr)
				err = tmperr
			}
			tmperr = ApplyParam(name, value)
			if tmperr != nil {
				LogError(tmperr)
				err = tmperr
			}
		}
		if err != nil {
			return "", err
		}
		return "", SaveCurrent()

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing name parameter")
		}
		return GlobalParams.Get(name)

	case "startprocess":
		return "", fmt.Errorf("executeEngineApi: startprocess is deprecated")
		// return "", TheProcessManager.StartRunning(process)

	case "stopprocess":
		return "", fmt.Errorf("executeEngineApi: stopprocess is deprecated")
		// process, ok := apiargs["process"]
		// if !ok {
		// 	return "", fmt.Errorf("executeEngineApi: missing process parameter")
		// }
		// err := TheProcessManager.KillProcess(process)
		// return "", err

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
		return e.StartRecording()

	case "stoprecording":
		return e.StopRecording()

	case "startplayback":
		fname, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing filename parameter")
		}
		return "", e.StartPlayback(fname)

	case "save":
		filename, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing filename parameter")
		}
		return "", GlobalParams.Save("engine", filename)

	case "done":
		e.SayDone() // needed for clean exit when profiling
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
		var x, y, z float32
		if !e.getFloat32(xs, &x) || !e.getFloat32(ys, &y) || !e.getFloat32(zs, &z) {
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

func ApplyParam(name string, value string) (err error) {
	var f float64
	var i int64

	// starting and stopping processes can't be done while loading initial parameters
	if strings.HasPrefix(name, "global.process.") {
		process := strings.TrimPrefix(name, "global.process.")
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

	switch name {

	case "global.attract":
		TheAttractManager.SetAttractMode(IsTrueValue(value))

	case "global.attractenabled":
		TheAttractManager.SetAttractEnabled(IsTrueValue(value))

	case "global.attractchecksecs":
		if TheEngine.getFloat(value, &f) {
			TheAttractManager.attractCheckSecs = f
		}
	case "global.attractchangeinterval":
		if TheEngine.getFloat(value, &f) {
			TheAttractManager.attractChangeInterval = f
		}
	case "global.attractgestureinterval":
		if TheEngine.getFloat(value, &f) {
			TheAttractManager.attractGestureInterval = f
		}
	case "global.attractidlesecs":
		if TheEngine.getInt(value, &i) {
			if i < 15 {
				LogWarn("global.attractidlesecs is too low, forcing to 15")
				i = 15
			}
			TheAttractManager.attractIdleSecs = float64(i)
		}
	case "global.looping_fadethreshold":
		if TheEngine.getFloat(value, &f) {
			TheCursorManager.LoopThreshold = float32(f)
		}

	case "global.looping_override":
		LogInfo("global.looping_override needs handling")

	case "global.looping_fade":
		LogInfo("global.looping_fade needs handling")

	case "global.looping_beats":
		LogInfo("global.looping_beats needs handling")

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
		if TheEngine.getInt(value, &i) {
			TheEngine.SetAutoTransposeBeats(int(i))
		}
	case "global.transpose":
		if TheEngine.getInt(value, &i) {
			TheEngine.SetTranspose(int(i))
		}
	case "global.log":
		SetLogTypes(value)

	case "global.midiinput":
		TheMidiIO.SetMidiInput(value)

	case "global.processchecksecs":
		if TheEngine.getFloat(value, &f) {
			TheProcessManager.processCheckSecs = f
		}

	case "global.nats":
		if IsTrueValue(value) {
			EngineSubscribeNats()
		} else {
			EngineCloseNats()
		}

	case "global.obsstream":
		if IsTrueValue(value) {
			err := ObsCommand("streamstart")
			LogIfError(err)
		} else {
			err := ObsCommand("streamstop")
			LogIfError(err)
		}

	case "global.emailto":

	case "global.twinsys":

	case "global.mmtt_zexpand":

	case "global.keykitallow":
	case "global.keykitoutput":

	case "global.timefret_low":
	case "global.timefret_mid":
	case "global.timefret_high":

	case "global.cursoroffsety":
	case "global.cursoroffsetx":

	case "global.gui":

	case "global.attractgestureduration":
	case "global.mmtt":
	case "global.testgesturentimes":
	case "global.testgesturenumsteps":
	case "global.title":
	case "global.scale":
	case "global.mmtt_xexpand":
	case "global.morphtype":
	case "global.testgestureinterval":
	case "global.bidulefile":
	case "global.emailpassword":
	case "global.cursorscaley":
	case "global.emaillogin":
	case "global.tempdir":
	case "global.attractimage":
	case "global.bidulepath":
	case "global.cursorscalex":
	case "global.midithrusynth":
	case "global.resolumepath":
	case "global.resolumetextlayer":
	case "global.keykitroot":
	case "global.keykitrun":
	case "global.guidefaultlevel":
	case "global.guisize":
	case "global.testgestureduration":
	case "global.winsize":
	case "global.notifygui":
	case "global.attractgesturenumsteps":
	case "global.keykitpath":
	case "global.mmtt_yexpand":
	case "global.helpimage":
	case "global.guishowall":

	case "global.attractnumbergesturesteps":
	case "global.attractguishow":
	case "global.attractsound":

	case "global.obspath":

	case "global.autostart":
		LogInfo("Engine.Set, obsolete parameter", "name", name, "value", value)

	default:
		if !strings.HasPrefix(name, "global.process.") {
			LogInfo("Engine.Set, unknown parameter", "name", name, "value", value)
		}
	}
	LogOfType("params", "Engine.ApplyParam", "name", name, "value", value)
	return nil
}

// func (e *Engine) SetParam(name string, value string) (err error) {
// 	LogOfType("params", "Engine.SetParam", "name", name, "value", value)
// 	return GlobalParams.SetParamWithString(name, value)
// }

func (e *Engine) executeSavedApi(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		category := optionalStringArg("category", apiargs, "*")
		return SavedList(category)
	default:
		LogWarn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecuteSavedApi unrecognized api=%s", api)
	}
}
