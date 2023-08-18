package hostwin

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"github.com/vizicist/palette/kit"
)

// ExecuteApi xxx
func (h HostWin) ExecuteApi(api string, apiargs map[string]string) (result string, err error) {

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
		return h.executeEngineApi(apisuffix, apiargs)
	case "saved":
		return h.executeSavedApi(apisuffix, apiargs)
	case "quadpro":
		if kit.TheQuadPro != nil {
			return kit.TheQuadPro.Api(apisuffix, apiargs)
		}
		return "", fmt.Errorf("no quadpro")
	case "patch":
		patchName := kit.ExtractAndRemoveValueOf("patch", apiargs)
		if patchName == "" {
			return "", fmt.Errorf("no patch value")
		}
		patch := kit.GetPatch(patchName)
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
func (h HostWin) ExecuteApiFromJson(rawjson string) (string, error) {
	args, err := kit.StringMap(rawjson)
	if err != nil {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: bad format of JSON")
	}
	api := kit.ExtractAndRemoveValueOf("api", args)
	if api == "" {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: no api value")
	}
	return h.ExecuteApi(api, args)
}

func (h HostWin) executeEngineApi(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "debugsched":
		return kit.TheScheduler.ToString(), nil

	case "debugpending":
		return kit.TheScheduler.PendingToString(), nil

	case "status":
		uptime := fmt.Sprintf("%f", kit.Uptime())
		attractmode := fmt.Sprintf("%v", kit.TheAttractManager.AttractModeIsOn())
		if kit.TheQuadPro == nil {
			result = kit.JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
			)
		} else {
			result = kit.JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
				"A", kit.Patchs["A"].Status(),
				"B", kit.Patchs["B"].Status(),
				"C", kit.Patchs["C"].Status(),
				"D", kit.Patchs["D"].Status(),
			)
		}
		return result, nil

	case "attract":
		v, ok := apiargs["onoff"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing onoff parameter")
		}
		kit.TheAttractManager.SetAttractMode(kit.IsTrueValue(v))
		return "", nil

	case "set":
		name, value, err := kit.GetNameValue(apiargs)
		if err != nil {
			return "", err
		}
		err = h.Set(name, value)
		if err != nil {
			return "", err
		}
		return "", kit.Params.Save("engine", "_Current")

	case "setparams":
		for name, value := range apiargs {
			e := h.Set(name, value)
			if e != nil {
				LogIfError(e)
				err = e
			}
		}
		if err != nil {
			return "", err
		}
		return "", kit.Params.Save("engine","_Current")

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing name parameter")
		}
		return kit.GetParam(name)

	case "startprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing process parameter")
		}
		return "", TheProcessManager.StartRunning(process)

	case "stopprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing process parameter")
		}
		err := TheProcessManager.KillProcess(process)
		return "", err

	case "showclip":
		s, ok := apiargs["clipnum"]
		if !ok {
			return "", fmt.Errorf("engine.showimage: missing filename parameter")
		}
		clipNum, err := strconv.Atoi(s)
		if err != nil {
			return "", fmt.Errorf("engine.showimage: bad clipnum parameter")
		}
		h.showClip(clipNum)
		return "", nil

	case "startrecording":
		// return kit.StartRecording()
		return "", fmt.Errorf("engine.startrecording: disabled")

	case "stoprecording":
		// return e.StopRecording()
		return "", fmt.Errorf("engine.stoprecording: disabled")

	case "startplayback":
		// fname, ok := apiargs["filename"]
		// if !ok {
		// 	return "", fmt.Errorf("executeEngineApi: missing filename parameter")
		// }
		// return "", h.StartPlayback(fname)
		return "", fmt.Errorf("engine.startplayback: disabled")

	case "save":
		filename, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("executeEngineApi: missing filename parameter")
		}
		return "", kit.Params.Save("engine", filename)

	case "done":
		h.SayDone() // needed for clean exit when profiling
		return "", nil

	case "audio_reset":
		go h.ResetAudio()

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
			kit.ChangeClicksPerSecond(float64(v))
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
		var pos kit.CursorPos
		xs := apiargs["x"]
		ys := apiargs["y"]
		zs := apiargs["z"]
		if xs == "" || ys == "" || zs == "" {
			return "", fmt.Errorf("playcursor: missing x, y, or z value")
		}
		var x, y, z float32
		if !h.getFloat32(xs, &x) || !h.getFloat32(ys, &y) || !h.getFloat32(zs, &z) {
			return "", fmt.Errorf("playcursor: bad x,y,z value")
		}
		pos = kit.CursorPos{X:x, Y:y, Z:z}
		go kit.TheCursorManager.PlayCursor(tag, dur, pos)
		return "", nil

	default:
		LogWarn("Router.ExecuteApi api is not recognized\n", "api", api)
		err = fmt.Errorf("ExecuteEngineApi: unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func (h HostWin) getFloat(value string, f *float64) bool {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = v
		return true
	}
}

func (h HostWin) getFloat32(value string, f *float32) bool {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = float32(v)
		return true
	}
}

func (h HostWin) getInt(value string, i *int64) bool {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*i = v
		return true
	}
}

func (h HostWin) Set(name string, value string) error {
	var f float64
	var i int64
	switch name {

	case "engine.attract":
		kit.TheAttractManager.SetAttractMode(kit.IsTrueValue(value))

	case "engine.attractenabled":
		kit.TheAttractManager.SetAttractEnabled(kit.IsTrueValue(value))

	case "engine.attractchecksecs":
		if h.getFloat(value, &f) {
			kit.TheAttractManager.AttractCheckSecs = f
		}
	case "engine.attractchangeinterval":
		if h.getFloat(value, &f) {
			kit.TheAttractManager.AttractChangeInterval = f
		}
	case "engine.attractgestureinterval":
		if h.getFloat(value, &f) {
			kit.TheAttractManager.AttractGestureInterval = f
		}
	case "engine.attractidlesecs":
		if h.getInt(value, &i) {
			if i < 10 {
				LogWarn("engine.attractidlesecs is too low, forcing to 10")
				i = 10
			}
			kit.TheAttractManager.AttractIdleSecs = float64(i)
		}
	case "engine.looping_fadethreshold":
		if h.getFloat(value, &f) {
			kit.TheCursorManager.LoopThreshold = float32(f)
		}

	case "engine.midithru":
		kit.Midithru = kit.IsTrueValue(value)

	case "engine.midisetexternalscale":
		kit.Midisetexternalscale = kit.IsTrueValue(value)

	case "engine.midithruscadjust":
		kit.MidiThruScadjust = kit.IsTrueValue(value)

	case "engine.oscoutput":
		h.oscoutput = kit.IsTrueValue(value)

	case "engine.autotranspose":
		kit.TheScheduler.AutoTransposeOn = kit.IsTrueValue(value)

	case "engine.autotransposebeats":
		if h.getInt(value, &i) {
			kit.TheScheduler.SetAutoTransposeBeats(int(i))
		}
	case "engine.transpose":
		if h.getInt(value, &i) {
			kit.TheScheduler.SetTranspose(int(i))
		}
	case "engine.log":
		SetLogTypes(value)

	case "engine.midiinput":
		TheMidiIO.SetMidiInput(value)

	case "engine.processchecksecs":
		if h.getFloat(value, &f) {
			TheProcessManager.processCheckSecs = f
		}
	}

	LogOfType("params", "Engine.Set", "name", name, "value", value)
	return kit.Params.Set(name, value)
}

func (h HostWin) executeSavedApi(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		return SavedList(apiargs)
	default:
		LogWarn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecuteSavedApi unrecognized api=%s", api)
	}
}
