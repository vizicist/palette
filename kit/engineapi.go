package kit

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	json "github.com/goccy/go-json"
)

// NoProcess when true, skip auto-starting processes
var NoProcess bool

// ExecuteAPI xxx
func ExecuteAPI(api string, apiargs map[string]string) (result string, err error) {

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
	case "global":
		return ExecuteGlobalAPI(apisuffix, apiargs)
	case "saved":
		return ExecuteSavedAPI(apisuffix, apiargs)
	case "quad":
		if theQuad != nil {
			return theQuad.API(apisuffix, apiargs)
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
		return patch.API(apisuffix, apiargs)
	case "cursor":
		return ExecuteCursorAPI(apisuffix, apiargs)
	case "stepper":
		return ExecuteStepperAPI(apisuffix, apiargs)
	default:
		return "", fmt.Errorf("unknown apitype: %s", apitype)
	}
	// unreachable
}

// ExecuteAPIFromJSON takes raw JSON (as a string of the form "{...}"") as an API and returns raw JSON
func ExecuteAPIFromJSON(rawjson string) (string, error) {

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in ExecuteAPIFromJson called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in ExecuteAPIFromJson has been called")
			LogError(err, "r", r, "stack", stacktrace)
		}
	}()

	args, err := StringMap(rawjson)
	if err != nil {
		return "", fmt.Errorf("Router.ExecuteAPIAsJson: bad format of JSON")
	}
	api := ExtractAndRemoveValueOf("api", args)
	if api == "" {
		return "", fmt.Errorf("Router.ExecuteAPIAsJson: no api value")
	}
	return ExecuteAPI(api, args)
}

type globalAPIHandler func(api string, apiargs map[string]string) (string, error)

var globalAPIHandlers = map[string]globalAPIHandler{
	"debugnil":             globalDebugNil,
	"debugsched":           globalDebugSched,
	"debugpending":         globalDebugPending,
	"status":               globalStatus,
	"attract":              globalAttract,
	"activate":             globalActivate,
	"load":                 globalLoad,
	"getboot":              globalGetBoot,
	"getbootwithprefix":    globalGetBoot,
	"setboot":              globalSetBoot,
	"setbootfromcurrent":   globalSetBootFromCurrent,
	"set":                  globalSet,
	"setparams":            globalSetParams,
	"get":                  globalGet,
	"getwithprefix":        globalGet,
	"showclip":             globalShowClip,
	"save":                 globalSave,
	"done":                 globalDone,
	"audio_reset":          globalAudioReset,
	"sendlogs":             globalSendLogs,
	"log":                  globalLog,
	"midi_midifile":        globalRemoved,
	"echo":                 globalEcho,
	"debug":                globalRemoved,
	"set_tempo_factor":     globalSetTempoFactor,
	"sampleplaybackreload": globalSamplePlaybackReload,
	"playcursor":           globalPlayCursor,
	"obsrecord":            globalObsRecord,
	"obsrecordstop":        globalObsRecordStop,
	"obssetup":             globalObsSetup,
}

func ExecuteGlobalAPI(api string, apiargs map[string]string) (string, error) {
	if api != "status" {
		LogOfType("api", "ExecuteGlobalAPI", "api", api, "apiargs", apiargs)
	}
	handler, ok := globalAPIHandlers[api]
	if !ok {
		LogWarn("ExecuteGlobalAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("ExecuteGlobalAPI: unrecognized api=%s", api)
	}
	return handler(api, apiargs)
}

func globalDebugNil(api string, apiargs map[string]string) (string, error) {
	// Generate a nil pointer panic
	var a *Engine
	a.SayDone()
	return "", nil // unreachable
}

func globalDebugSched(api string, apiargs map[string]string) (string, error) {
	return theScheduler.ToString(), nil
}

func globalDebugPending(api string, apiargs map[string]string) (string, error) {
	return theScheduler.PendingToString(), nil
}

func globalStatus(api string, apiargs map[string]string) (string, error) {
	bytes, err := json.Marshal(uiStatusSnapshot())
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func globalAttract(api string, apiargs map[string]string) (string, error) {
	v, err := needStringArg("onoff", api, apiargs)
	if err != nil {
		return "", err
	}
	theAttractManager.SetAttractMode(IsTrueValue(v))
	NotifyStatusChanged()
	return "", nil
}

func globalActivate(api string, apiargs map[string]string) (string, error) {
	process, err := needStringArg("process", api, apiargs)
	if err != nil {
		return "", err
	}
	return "", ActivateProcess(process)
}

func globalLoad(api string, apiargs map[string]string) (string, error) {
	fname, err := needStringArg("filename", api, apiargs)
	if err != nil {
		return "", err
	}
	err = LoadGlobalParamsFrom(fname, true)
	if err == nil {
		rememberPresetSelection("global", "*", fname)
	}
	return "", err
}

// globalParamGet reads name from params, returning all matching parameters when
// the api name ends in "withprefix" and a single value otherwise.
func globalParamGet(params *ParamValues, api string, name string) (string, error) {
	if strings.HasSuffix(api, "withprefix") {
		return params.GetWithPrefix(name)
	}
	return params.Get(name)
}

func globalGetBoot(api string, apiargs map[string]string) (string, error) {
	name, err := needStringArg("name", api, apiargs)
	if err != nil {
		return "", err
	}
	name = canonicalGlobalParamName(name)
	params, err := LoadBootParamValues()
	if err != nil {
		return "", err
	}
	return globalParamGet(params, api, name)
}

func globalSetBoot(api string, apiargs map[string]string) (string, error) {
	name, value, err := GetNameValue(apiargs)
	if err != nil {
		return "", err
	}
	name = canonicalGlobalParamName(name)
	params, err := LoadBootParamValues()
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
}

func globalSetBootFromCurrent(api string, apiargs map[string]string) (string, error) {
	err := GlobalParams.Save("global", "_Boot")
	if err != nil {
		return "", err
	}
	// Refresh EVERYTHING from _Boot, as if rebooting
	return "", LoadGlobalParamsFrom("_Boot", true)
}

func globalSet(api string, apiargs map[string]string) (string, error) {
	name, value, err := GetNameValue(apiargs)
	if err != nil {
		return "", err
	}
	err = SetAndApplyGlobalParam(name, value)
	if err != nil {
		return "", err
	}
	return "", SaveCurrentGlobalParams()
}

func globalSetParams(api string, apiargs map[string]string) (string, error) {
	var err error
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
}

func globalGet(api string, apiargs map[string]string) (string, error) {
	name, err := needStringArg("name", api, apiargs)
	if err != nil {
		return "", err
	}
	name = canonicalGlobalParamName(name)
	return globalParamGet(GlobalParams, api, name)
}

func globalShowClip(api string, apiargs map[string]string) (string, error) {
	s, err := needStringArg("clipnum", api, apiargs)
	if err != nil {
		return "", err
	}
	clipNum, err := strconv.Atoi(s)
	if err != nil {
		return "", fmt.Errorf("global.showclip: bad clipnum value: %w", err)
	}
	TheResolume().showTextLayerClip(clipNum)
	return "", nil
}

func globalObsRecord(api string, apiargs map[string]string) (string, error) {
	return ObsRecordClip()
}

func globalObsRecordStop(api string, apiargs map[string]string) (string, error) {
	return ObsRecordStop()
}

func globalObsSetup(api string, apiargs map[string]string) (string, error) {
	err := ObsAutoSetup()
	if err != nil {
		return "", err
	}
	return `{"status":"ok"}`, nil
}

func globalSave(api string, apiargs map[string]string) (string, error) {
	filename, err := needStringArg("filename", api, apiargs)
	if err != nil {
		return "", err
	}
	return "", GlobalParams.Save("global", filename)
}

func globalDone(api string, apiargs map[string]string) (string, error) {
	theEngine.SayDone() // needed for clean exit when profiling
	return "", nil
}

func globalAudioReset(api string, apiargs map[string]string) (string, error) {
	go TheBidule().Reset()
	return "", nil
}

func globalSendLogs(api string, apiargs map[string]string) (string, error) {
	return "", ArchiveLogs()
}

func globalLog(api string, apiargs map[string]string) (string, error) {
	// Parse optional file parameter (defaults to engine.log)
	// Sanitize to basename only for security (no path traversal)
	logfile := "engine.log"
	if fileStr, ok := apiargs["file"]; ok && fileStr != "" {
		logfile = filepath.Base(fileStr)
	}

	// Parse optional time range parameters
	var startTime, endTime *time.Time
	if startStr, ok := apiargs["start"]; ok && startStr != "" {
		t, err := time.Parse(PaletteTimeLayout, startStr)
		if err != nil {
			return "", fmt.Errorf("invalid start time format, use RFC3339: %w", err)
		}
		startTime = &t
	}
	if endStr, ok := apiargs["end"]; ok && endStr != "" {
		t, err := time.Parse(PaletteTimeLayout, endStr)
		if err != nil {
			return "", fmt.Errorf("invalid end time format, use RFC3339: %w", err)
		}
		endTime = &t
	}

	// Parse optional limit and offset
	limit := 500
	if limitStr, ok := apiargs["limit"]; ok && limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			return "", fmt.Errorf("invalid limit value: %w", err)
		}
		limit = l
	}
	offset := 0
	if offsetStr, ok := apiargs["offset"]; ok && offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err != nil {
			return "", fmt.Errorf("invalid offset value: %w", err)
		}
		offset = o
	}

	entries, err := ReadLogEntries(logfile, startTime, endTime, limit, offset)
	if err != nil {
		return "", err
	}
	jsonBytes, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("failed to marshal log entries: %w", err)
	}
	return string(jsonBytes), nil
}

func globalRemoved(api string, apiargs map[string]string) (string, error) {
	return "", fmt.Errorf("%s API has been removed", api)
}

func globalEcho(api string, apiargs map[string]string) (string, error) {
	value, ok := apiargs["value"]
	if !ok {
		value = "ECHO!"
	}
	return value, nil
}

func globalSetTempoFactor(api string, apiargs map[string]string) (string, error) {
	v, err := needFloatArg("value", api, apiargs)
	if err != nil {
		return "", err
	}
	ChangeClicksPerSecond(float64(v))
	NotifyStepperChanged()
	return "", nil
}

func globalSamplePlaybackReload(api string, apiargs map[string]string) (string, error) {
	if err := ReloadSamplePlaybackServiceSamples(); err != nil {
		return "", err
	}
	return JSONObject("ok", "true"), nil
}

func globalPlayCursor(api string, apiargs map[string]string) (string, error) {
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
	pos := CursorPos{x, y, z}
	go theCursorManager.PlayCursor(tag, dur, pos)
	return "", nil
}

func GetFloat(value string, f *float64) bool {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		LogIfError(err)
		return false
	}
	*f = v
	return true
}

func GetInt(value string, i *int64) bool {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		LogIfError(err)
		return false
	}
	*i = v
	return true
}

func ActivateGlobalParam(name string) {
	name = canonicalGlobalParamName(name)
	val, err := GetParam(name)
	LogIfError(err)
	err = SetAndApplyGlobalParam(name, val)
	LogIfError(err)

}

func SetAndApplyGlobalParam(name string, value string) (err error) {
	name = canonicalGlobalParamName(name)
	err = GlobalParams.SetParamWithString(name, value)
	if err != nil {
		return err
	}
	err = ApplyGlobalParam(name, value)
	if err == nil {
		NotifyStatusChanged()
	}
	return err
}

var legacyGlobalParamAliases = map[string]string{
	"global.initialpage": "global.mode",
}

func canonicalGlobalParamName(name string) string {
	if canonical, ok := legacyGlobalParamAliases[name]; ok {
		return canonical
	}
	return name
}

var attractFloatParamSetters = map[string]func(*AttractManager, float64){
	"global.attractgestureinterval":      func(am *AttractManager, f float64) { am.GestureInterval = f },
	"global.attractpresetchangeinterval": func(am *AttractManager, f float64) { am.PresetChangeInterval = f },
	"global.attractgestureduration":      func(am *AttractManager, f float64) { am.GestureDuration = f },
	"global.attractgestureminlength":     func(am *AttractManager, f float64) { am.GestureMinLength = f },
	"global.attractgesturemaxlength":     func(am *AttractManager, f float64) { am.GestureMaxLength = f },
	"global.attractgesturezmin":          func(am *AttractManager, f float64) { am.GestureZMin = f },
	"global.attractgesturezmax":          func(am *AttractManager, f float64) { am.GestureZMax = f },
}

func applyAttractGlobalParam(name string, value string) bool {
	if theAttractManager == nil {
		return false
	}
	switch name {
	case "global.attract":
		theAttractManager.SetAttractMode(IsTrueValue(value))
		return true
	case "global.attractenabled":
		theAttractManager.SetAttractEnabled(IsTrueValue(value))
		return true
	case "global.attractidlesecs":
		var i int64
		if GetInt(value, &i) {
			if i < 15 {
				LogWarn("global.attractidlesecs is too low, forcing to 15")
				i = 15
			}
			theAttractManager.IdleSecs = float64(i)
		}
		return true
	case "global.attractgesturenumsteps":
		var i int64
		if GetInt(value, &i) {
			if i < 1 {
				i = 1
			}
			theAttractManager.GestureNumSteps = int(i)
		}
		return true
	}

	setter, ok := attractFloatParamSetters[name]
	if !ok {
		return false
	}
	var f float64
	if GetFloat(value, &f) {
		setter(theAttractManager, f)
	}
	return true
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
		process := strings.TrimPrefix(name, "global.process.")
		if theProcessManager.IsAvailable(process) {
			if effectiveProcessRunit(process, IsTrueValue(value)) {
				err = theProcessManager.StartRunning(process)
			} else {
				err = theProcessManager.StopRunning(process)
			}
			if err != nil {
				LogError(err)
				return err
			}
		}
	}
	if name == "global.resolumepath" && theProcessManager != nil {
		theProcessManager.AddProcessBuiltIn("resolume")
	}
	if applyAttractGlobalParam(name, value) {
		return nil
	}

	switch name {
	case "global.erae":
		b, _ := GetParamBool("global.erae")
		if b {
			theErae.EraeAPIModeEnable()
			theMidiIO.SetMidiInput("Erae 2")
		}

	case "global.looping_fadethreshold":
		if GetFloat(value, &f) {
			theCursorManager.LoopThreshold = f
		}

	case "global.mode":
		mode := normalizeMode(value)
		if mode != value {
			if err := GlobalParams.SetParamWithString(name, mode); err != nil {
				LogError(err)
				return err
			}
		}
		ApplyMode()

	case "global.sampleplaybackcompressed":
		SetSamplePlaybackServiceCompressed(IsTrueValue(value))

	case "global.sampleplaybackreverb":
		wet := 0.0
		if GetFloat(value, &wet) {
			SetSamplePlaybackServiceReverbWet(clampSamplePlaybackReverbWet(wet))
		}

	case "global.sampleplaybackreverblength":
		length := defaultSamplePlaybackReverbLength
		if GetFloat(value, &length) {
			SetSamplePlaybackServiceReverbLength(clampSamplePlaybackReverbLength(length))
		}

	case "global.sampleplaybackwords":
		words := int64(2)
		if GetInt(value, &words) {
			SetSamplePlaybackServiceWords(clampSamplePlaybackWords(int(words)))
			if err := ReloadSamplePlaybackServiceSamples(); err != nil {
				LogWarn("global.sampleplaybackwords reload failed", "err", err)
			}
		}

	case "global.sampleplaybackminmp3seconds":
		seconds := defaultSamplePlaybackMinimumMP3DurationSecs
		if GetFloat(value, &seconds) {
			SetSamplePlaybackServiceMinimumMP3Duration(clampSamplePlaybackMinimumMP3Duration(seconds))
			if err := ReloadSamplePlaybackServiceSamples(); err != nil {
				LogWarn("global.sampleplaybackminmp3seconds reload failed", "err", err)
			}
		}

	case "global.midithru":
		theRouter.midithru = IsTrueValue(value)

	case "global.midisetexternalscale":
		theRouter.midisetexternalscale = IsTrueValue(value)

	case "global.midithruscadjust":
		theRouter.midiThruScadjust = IsTrueValue(value)

	case "global.oscoutput":
		theEngine.oscoutput = IsTrueValue(value)

	case "global.autotranspose":
		theEngine.autoTransposeOn = IsTrueValue(value)

	case "global.autotransposebeats":
		if GetInt(value, &i) {
			theEngine.SetAutoTransposeBeats(int(i))
		}
	case "global.transpose":
		if GetInt(value, &i) {
			theEngine.SetTranspose(int(i))
		}
	case "global.log":
		SetLogTypes(value)

	case "global.midiinput":
		theMidiIO.SetMidiInput(value)

	case "global.processchecksecs":
		if GetFloat(value, &f) {
			theProcessManager.processCheckSecs = f
		}

	case "global.obsstream":
		// Only send OBS commands if OBS is actually running
		running, err := IsRunning("obs")
		if err == nil && running {
			if IsTrueValue(value) {
				err := ObsCommand("streamstart")
				LogIfError(err)
			} else {
				_ = ObsCommand("streamstop")
			}
		}
	}
	return nil
}

func ExecuteSavedAPI(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		category := optionalStringArg("category", apiargs, "*")
		return SavedListAsString(category)
	case "remove":
		category, err := needStringArg("category", "saved.remove", apiargs)
		if err != nil {
			return "", err
		}
		filename, err := needStringArg("filename", "saved.remove", apiargs)
		if err != nil {
			return "", err
		}
		return "", RemoveSavedFile(category, filename)
	case "paramdefs":
		category := optionalStringArg("category", apiargs, "*")
		return ParamDefsForCategory(category)
	case "paraminits":
		category := optionalStringArg("category", apiargs, "*")
		return ParamInitValuesForCategory(category)
	case "paramrands":
		category := optionalStringArg("category", apiargs, "*")
		return ParamRandomValuesForCategory(category)
	case "paramenums":
		return ParamEnumsAsJSON()
	case "paramdefsjson":
		return ParamDefsAsJSON()
	default:
		LogWarn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecuteSavedAPI unrecognized api=%s", api)
	}
}
