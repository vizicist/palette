package plugin

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

type PalettePro struct {

	// Per-layer things
	layer      map[string]*engine.Layer
	layerLogic map[string]*LayerLogic

	// PalettePro parameters
	miscparams   *engine.ParamValues
	globalparams *engine.ParamValues

	started        bool
	midiinputlayer *engine.Layer

	generateVisuals bool
	generateSound   bool

	MIDIOctaveShift  int
	MIDINumDown      int
	MIDIThru         bool
	MIDIThruScadjust bool
	MIDISetScale     bool
	MIDIQuantized    bool
	TransposePitch   int
	externalScale    *engine.Scale
	nextCursorNum    int

	attractModeIsOn         bool
	lastAttractModeChange   time.Time
	lastAttractGestureTime  time.Time
	lastAttractPresetChange time.Time

	// parameters
	attractGestureInterval float64
	attractPresetInterval  float64
	attractPreset          string

	lastAttractCheck time.Time
	lastProcessCheck time.Time

	attractCheckSecs float64
	attractIdleSecs  float64
	processCheckSecs float64

	transposeAuto   bool
	transposeNext   engine.Clicks
	transposeClicks engine.Clicks // time between auto transpose changes
	transposeIndex  int           // current place in tranposeValues
	transposeValues []int
}

func init() {
	ppro := &PalettePro{
		layer:        map[string]*engine.Layer{},
		layerLogic:   map[string]*LayerLogic{},
		miscparams:   engine.NewParamValues(),
		globalparams: engine.NewParamValues(),

		attractModeIsOn: false,
		generateVisuals: true,
		generateSound:   true,

		lastAttractModeChange:   time.Now(),
		lastAttractGestureTime:  time.Now(),
		lastAttractPresetChange: time.Now(),
		lastAttractCheck:        time.Now(),
		lastProcessCheck:        time.Time{},

		attractGestureInterval: 0,
		attractPresetInterval:  0,
		attractPreset:          "",
		attractCheckSecs:       0,
		attractIdleSecs:        0, // default is no attract mode

		processCheckSecs: 0,

		transposeAuto:   true,
		transposeNext:   0,
		transposeClicks: 0,
		transposeIndex:  0,
		transposeValues: []int{0, -2, 3, -5},
	}
	engine.RegisterPlugin("ppro", ppro.Api)
}

func (ppro *PalettePro) Api(ctx *engine.PluginContext, api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "start":
		return "", ppro.start(ctx)

	case "stop":
		ppro.started = false
		return "", ctx.StopRunning("all")

	case "set":
		return ppro.onSet(ctx, apiargs)

	case "get":
		return ppro.onGet(apiargs)

	case "event":
		eventName, ok := apiargs["event"]
		if !ok {
			return "", fmt.Errorf("PalettePro: Missing event argument")
		}
		if eventName == "click" {
			return ppro.onClick(ctx, apiargs)
		}

		// engine.LogInfo("Ppro.event", "eventName", eventName)
		switch eventName {
		case "midi":
			return ppro.onMidiEvent(ctx, apiargs)
		case "cursor":
			return ppro.onCursorEvent(ctx, apiargs)
		case "clientrestart":
			return ppro.onClientRestart(ctx, apiargs)
		case "notification_of_layer_set":
			return ppro.onLayerSet(ctx, apiargs)
		case "notification_of_layer_refresh_all":
			return ppro.onLayerRefreshAll(ctx, apiargs)
		case "presetsave":
			// Events from listeners in the Layers
			// will trigger saving of the entire Preset
			return "", ppro.presetSave("_Current")
		default:
			return "", fmt.Errorf("PalettePro: Unhandled event type %s", eventName)
		}

	case "ANO":
		for _, layer := range ppro.layer {
			layer.Synth.SendANO()
		}
		return "", nil

	case "attract":
		onoffstr, ok := apiargs["onoff"]
		if !ok {
			return "", fmt.Errorf("missing onoff parameter")
		}
		onoff, err := engine.IsTrueValue(onoffstr)
		if err != nil {
			return "", err
		}
		ppro.setAttractMode(onoff)
		return "", nil

	case "clearexternalscale":
		ppro.clearExternalScale()
		return "", nil

	case "echo":
		value, ok := apiargs["value"]
		if !ok {
			value = "ECHO!"
		}
		return value, nil

	case "load":
		category, oksaved := apiargs["category"]
		if !oksaved {
			return "", fmt.Errorf("missing category parameter")
		}
		filename, oksaved := apiargs["filename"]
		if !oksaved {
			return "", fmt.Errorf("missing filename parameter")
		}
		ppro.setAttractMode(false)
		return "", ppro.Load(ctx, category, filename)

	case "save":
		category, oksaved := apiargs["category"]
		if !oksaved {
			return "", fmt.Errorf("missing category parameter")
		}
		filename, oksaved := apiargs["filename"]
		if !oksaved {
			return "", fmt.Errorf("missing filename parameter")
		}
		return "", ppro.save(category, filename)

	case "startprocess":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("ExecuteAPI: missing process argument")
		} else {
			err = ctx.StartRunning(process)
		}
		return "", err

	case "stopprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("ExecuteAPI: missing process argument")
		}
		return "", ctx.StopRunning(process)

	case "status":
		result = engine.JsonObject(
			"uptime", fmt.Sprintf("%f", ctx.Uptime()),
			"attractmode", fmt.Sprintf("%v", ppro.attractModeIsOn),
		)
		return result, nil

	case "test":
		var ntimes = 10                 // default
		var dt = 500 * time.Millisecond // default
		s, ok := apiargs["ntimes"]
		if ok {
			tmp, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return "", err
			}
			ntimes = int(tmp)
		}
		s, ok = apiargs["dt"]
		if ok {
			tmp, err := time.ParseDuration(s)
			if err != nil {
				return "", err
			}
			dt = tmp
		}
		go ppro.doTest(ctx, ntimes, dt)
		return "", nil

	default:
		engine.LogWarn("PalettePro.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("PalettePro.Api unrecognized api=%s", api)
	}
}

func (ppro *PalettePro) start(ctx *engine.PluginContext) error {

	if ppro.started {
		return fmt.Errorf("PalettePro: already started")
	}
	ppro.started = true

	ppro.clearExternalScale()
	for nm, pd := range engine.ParamDefs {
		if pd.Category == "misc" {
			err := ppro.miscparams.SetParamValueWithString(nm, pd.Init)
			if err != nil {
				engine.LogError(err)
			}
		}
		if pd.Category == "global" {
			err := ppro.globalparams.SetParamValueWithString(nm, pd.Init)
			if err != nil {
				engine.LogError(err)
			}
		}
	}

	transposebeats := engine.Clicks(engine.ConfigIntWithDefault("transposebeats", 48))
	ppro.transposeNext = transposebeats * engine.OneBeat
	ppro.transposeClicks = transposebeats

	ppro.generateVisuals = engine.ConfigBoolWithDefault("generatevisuals", true)
	ppro.generateSound = engine.ConfigBoolWithDefault("generatesound", true)
	ppro.transposeAuto = engine.ConfigBoolWithDefault("transposeauto", true)

	ctx.AddProcessBuiltIn("resolume")
	ctx.AddProcessBuiltIn("bidule")
	ctx.AddProcess("gui", ppro.guiInfo())
	ctx.AddProcess("mmtt", ppro.mmttInfo())

	ctx.AllowSource("A", "B", "C", "D")

	_ = ppro.addLayer(ctx, "A")
	_ = ppro.addLayer(ctx, "B")
	_ = ppro.addLayer(ctx, "C")
	_ = ppro.addLayer(ctx, "D")

	engine.LogError(ppro.Load(ctx, "global", "_Current"))
	engine.LogError(ppro.Load(ctx, "preset", "_Current"))

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	ppro.processCheckSecs = engine.ConfigFloatWithDefault("processchecksecs", 60)

	ppro.attractCheckSecs = engine.ConfigFloatWithDefault("attractchecksecs", 2)
	ppro.attractIdleSecs = engine.ConfigFloatWithDefault("attractidlesecs", 0)

	ppro.attractPresetInterval = engine.ConfigFloatWithDefault("attractpresetinterval", 30)
	ppro.attractGestureInterval = engine.ConfigFloatWithDefault("attractgestureinterval", 0.5)

	ppro.attractPreset = engine.ConfigStringWithDefault("attractpreset", "random")

	return nil
}

func (ppro *PalettePro) onCursorEvent(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {

	ddu, ok := apiargs["ddu"]
	if !ok {
		return "", fmt.Errorf("PalettePro: Missing ddu argument")
	}

	if ddu == "clear" {
		ctx.ClearCursors()
		return "", nil
	}

	x, y, z, err := ctx.GetArgsXYZ(apiargs)
	if err != nil {
		return "", err
	}

	// source, ok := apiargs["source"]
	// if !ok {
	// 	source = ""
	// }

	cid, ok := apiargs["cid"]
	if !ok {
		cid = ""
	}

	ce := engine.CursorEvent{
		Cid:   cid,
		Click: 0,
		Ddu:   ddu,
		X:     x,
		Y:     y,
		Z:     z,
		Area:  0,
	}

	// Any non-internal cursor will turn attract mode off.
	if !ce.IsInternal() {
		ppro.setAttractMode(false)
	}

	// For the moment, the cursor to layerLogic mapping is 1-to-1.
	// I.e. ce.Source of "A" maps to layerLogic "A"
	layerLogic, ok := ppro.layerLogic[ce.Source()]
	if !ok {
		return "", nil
	}
	if layerLogic != nil {
		if ppro.generateSound && !ppro.attractModeIsOn {
			if ce.Ddu != "drag" {
				layerLogic.generateSoundFromCursor(ctx, ce)
			}
		}
		if ppro.generateVisuals {
			layerLogic.generateVisualsFromCursor(ce)
		}
	}
	return "", nil
}

func (ppro *PalettePro) setAttractMode(onoff bool) {

	if onoff == ppro.attractModeIsOn {
		return // already in that mode
	}
	// Throttle it a bit
	secondsSince := time.Since(ppro.lastAttractModeChange).Seconds()
	if secondsSince > 1.0 {
		engine.LogOfType("attract", "PalettePro: changing attract", "onoff", onoff)
		ppro.attractModeIsOn = onoff
		ppro.lastAttractModeChange = time.Now()
	}
}

func (ppro *PalettePro) clearExternalScale() {
	ppro.externalScale = engine.MakeScale()
}

func (ppro *PalettePro) onClientRestart(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {
	// These are messages from the Palette FFGL plugins in Resolume,
	// telling us that they have restarted, and we should resend all parameters
	portnum, ok := apiargs["portnum"]
	if !ok {
		return "", fmt.Errorf("PalettePro: Missing portnum argument")
	}
	ffglportnum, err := strconv.Atoi(portnum)
	if err != nil {
		return "", err
	}
	engine.LogInfo("ppro got clientrestart", "portnum", portnum)
	// The restart message contains the
	// portnum of the plugin that restarted.
	for _, layer := range ppro.layer {
		layer.RefreshAllIfPortnumMatches(ffglportnum)
	}
	return "", nil
}

// onSet is only used for parameters in a Preset that are not in its layers.
// NOTE: this routine does not automatically persist the values to disk.
func (ppro *PalettePro) onSet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {

	paramName, paramValue, err := engine.GetNameValue(apiargs)
	if err != nil {
		return "", fmt.Errorf("PalettePro.onSet: %s", err)
	}

	switch {

	case strings.HasPrefix(paramName, "misc."):

		if paramName == "misc.midiinputlayer" {
			layer := ctx.GetLayer(paramValue)
			if layer == nil {
				return "", fmt.Errorf("PalettePro.onSet: misc.midiinputlayer value is not a layer %s", paramValue)
			}
			ppro.midiinputlayer = layer
		}

		return "", ppro.miscparams.Set(paramName, paramValue)

	case strings.HasPrefix(paramName, "global"):
		return "", ppro.globalparams.Set(paramName, paramValue)

	default:
		return "", fmt.Errorf("PalettePro.onSet: can't handle parameter %s", paramName)
	}
}

func (ppro *PalettePro) onGet(apiargs map[string]string) (result string, err error) {
	paramName, ok := apiargs["name"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerSet: Missing name argument")
	}
	if strings.HasPrefix(paramName, "misc") {
		return ppro.miscparams.Get(paramName), nil
	} else if strings.HasPrefix(paramName, "global") {
		return ppro.globalparams.Get(paramName), nil
	} else {
		return "", fmt.Errorf("PalettePro.onGet: can't handle parameter %s", paramName)
	}
}

func (ppro *PalettePro) onLayerRefreshAll(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {
	layerName, ok := apiargs["layer"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerRefreshAll: Missing layer argument")
	}
	layer, ok := ppro.layer[layerName]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerRefreshAll: No layer named %s", layerName)
	}
	ppro.refreshAllLayerValues(ctx, layer)
	return "", nil
}

func (ppro *PalettePro) refreshAllLayerValues(ctx *engine.PluginContext, layer *engine.Layer) {
	for _, paramName := range layer.ParamNames() {
		paramValue := layer.Get(paramName)
		ppro.refreshLayerValue(ctx, layer, paramName, paramValue)
	}
}

func (ppro *PalettePro) onLayerSet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {
	layerName, ok := apiargs["layer"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerSet: Missing layer argument")
	}
	paramName, paramValue, err := engine.GetNameValue(apiargs)
	if err != nil {
		return "", fmt.Errorf("PalettePro.onLayerSet: err=%s", err)
	}
	layer, ok := ppro.layer[layerName]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerSet: No layer named %s", layerName)
	}
	ppro.refreshLayerValue(ctx, layer, paramName, paramValue)
	return "", nil
}

func (ppro *PalettePro) refreshLayerValue(ctx *engine.PluginContext, layer *engine.Layer, paramName string, paramValue string) {

	if strings.HasPrefix(paramName, "visual.") {
		name := strings.TrimPrefix(paramName, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, paramValue)
		msg.Append(args)
		engine.TheResolume().ToFreeFramePlugin(layer.Name(), msg)
	}

	if strings.HasPrefix(paramName, "effect.") {
		name := strings.TrimPrefix(paramName, "effect.")
		// Effect parameters get sent to Resolume
		engine.TheResolume().SendEffectParam(layer.Name(), name, paramValue)
	}

	if paramName == "sound.synth" {
		synth := ctx.GetSynth(paramValue)
		if synth == nil {
			engine.LogWarn("PalettePro: no synth named", "synth", paramValue)
		}
		layer.Synth = synth
	}
}

func (ppro *PalettePro) onClick(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {
	ppro.checkAttract(ctx)
	ppro.checkProcess(ctx)
	return "", nil
}

func (ppro *PalettePro) checkProcess(ctx *engine.PluginContext) {

	firstTime := (ppro.lastProcessCheck == time.Time{})

	// If processCheckSecs is 0, process checking is disabled
	processCheckEnabled := ppro.processCheckSecs > 0
	if !processCheckEnabled {
		if firstTime {
			engine.LogInfo("Process Checking is disabled.")
		}
		return
	}

	now := time.Now()
	sinceLastProcessCheck := now.Sub(ppro.lastProcessCheck).Seconds()
	if processCheckEnabled && (firstTime || sinceLastProcessCheck > ppro.processCheckSecs) {
		// Put it in background, so calling
		// tasklist or ps doesn't disrupt realtime
		// The sleep here is because if you kill something and
		// immediately check to see if it's running, it reports that
		// it's stil running.
		go func() {
			time.Sleep(2 * time.Second)
			ctx.CheckAutostartProcesses()
		}()
		ppro.lastProcessCheck = now
	}
}

func (ppro *PalettePro) checkAttract(ctx *engine.PluginContext) {

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()
	attractModeEnabled := ppro.attractIdleSecs > -1
	sinceLastAttractCheck := now.Sub(ppro.lastAttractCheck).Seconds()
	if attractModeEnabled && sinceLastAttractCheck > ppro.attractCheckSecs {
		ppro.lastAttractCheck = now
		// There's a delay when checking cursor activity to turn attract mod on.
		// Non-internal cursor activity turns attract mode off instantly.
		sinceLastAttractModeChange := time.Since(ppro.lastAttractModeChange).Seconds()
		if !ppro.attractModeIsOn && sinceLastAttractModeChange > ppro.attractIdleSecs {
			// Nothing happening for a while, turn attract mode on
			ppro.setAttractMode(true)
		}
	}

	if ppro.attractModeIsOn {
		ppro.doAttractAction(ctx)
	}
}

func (ppro *PalettePro) onMidiEvent(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {

	me, err := engine.MidiEventFromMap(apiargs)
	if err != nil {
		return "", err
	}

	engine.LogInfo("PalettePro.onMidiEvent", "me", me)

	/*
		if EraeEnabled {
			HandleEraeMIDI(me)
		}
	*/
	layerName := ppro.miscparams.GetStringValue("misc.midiinputlayer", "A")
	layer := ctx.GetLayer(layerName)

	if ppro.MIDIThru {
		engine.LogInfo("PassThruMIDI", "msg", me.Msg)
		ctx.ScheduleMidi(layer, me.Msg, ctx.CurrentClick())
	}
	if ppro.MIDISetScale {
		ppro.handleMIDISetScaleNote(me)
	}

	return "", nil
}

func (ppro *PalettePro) doTest(ctx *engine.PluginContext, ntimes int, dt time.Duration) {
	engine.LogInfo("doTest start", "ntimes", ntimes, "dt", dt)
	for n := 0; n < ntimes; n++ {
		if n > 0 {
			time.Sleep(dt)
		}
		source := string("ABCD"[rand.Int()%4])
		cid := source + "#0"
		thismap := map[string]string{
			"cid":   cid,
			"event": "cursor",
			"ddu":   "down",
			"x":     fmt.Sprintf("%f", rand.Float32()),
			"y":     fmt.Sprintf("%f", rand.Float32()),
			"z":     fmt.Sprintf("%f", rand.Float32()),
		}
		_, err := ppro.Api(ctx, "event", thismap)
		if err != nil {
			engine.LogError(err)
			return
		}
		time.Sleep(dt)
		thismap = map[string]string{
			"cid":   cid,
			"event": "cursor",
			"ddu":   "up",
			"x":     fmt.Sprintf("%f", rand.Float32()),
			"y":     fmt.Sprintf("%f", rand.Float32()),
			"z":     fmt.Sprintf("%f", rand.Float32()),
		}
		_, err = ppro.Api(ctx, "event", thismap)
		if err != nil {
			engine.LogError(err)
			return
		}
	}
	engine.LogInfo("doTest end")
}

func (ppro *PalettePro) loadPresetRand(ctx *engine.PluginContext) {

	arr, err := engine.SavedFileList("preset")
	if err != nil {
		engine.LogError(err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	engine.LogInfo("loadPresetRand", "preset", arr[rn])

	engine.LogError(ppro.Load(ctx, "preset", arr[rn]))

	for _, layer := range ppro.layer {
		layer.AlertListenersToRefreshAll()
	}

	if err != nil {
		engine.LogError(err)
	}
}

func (ppro *PalettePro) Load(ctx *engine.PluginContext, category string, filename string) error {

	path := engine.SavedFilePath(category, filename)
	paramsMap, err := engine.LoadParamsMap(path)
	if err != nil {
		engine.LogError(err)
		return err
	}

	engine.LogInfo("PalettePro.Load", "category", category, "filename", filename)

	var lasterr error

	switch category {
	case "global":
		ppro.globalparams.ApplyParamsTo("global", paramsMap)

	case "misc":
		ppro.miscparams.ApplyParamsTo("misc", paramsMap)

	case "preset":
		// The Preset has one copy of misc.* params, and 4 layers of sound/visual/effect params
		for nm, val := range paramsMap {
			valstr, ok := val.(string)
			if !ok {
				engine.LogWarn("non-string value?", "nm", nm, "val", val)
				continue
			}
			if strings.HasPrefix(nm, "misc.") {
				engine.LogError(ppro.miscparams.Set(nm, valstr))
			}
		}

		for _, layer := range ppro.layer {
			err := layer.ApplyPresetTo(paramsMap)
			if err != nil {
				engine.LogError(err)
				lasterr = err
			}
			ppro.refreshAllLayerValues(ctx, layer)
		}

	case "layer", "sound", "visual", "effect":
		engine.LogWarn("PalettePro.Load: unhandled", "category", category, "filename", filename)
	default:
		engine.LogWarn("PalettePro.Load: other unhandled", "category", category, "filename", filename)
	}

	// Decide what _Current things we should save
	// when we load something.  E.g. if we're
	// loading a layer with something, we want to
	// then save the entire preset in preset._Current
	err = nil
	switch category {
	case "global":
		if filename == "_Current" {
			err = ppro.save("global", "_Current")
		}
	case "layer", "sound", "visual", "effect":
		if filename != "_Current" {
			err = ppro.presetSave("_Current")
		}
	}
	if err != nil {
		engine.LogError(err)
		lasterr = err
	}
	return lasterr
}

func (ppro *PalettePro) save(category string, filename string) (err error) {

	engine.LogInfo("PalettePro.Save", "category", category, "filename", filename)

	if category == "misc" {
		err = ppro.miscparams.Save("misc", filename)
	} else if category == "global" {
		err = ppro.globalparams.Save("global", filename)
	} else if category == "preset" {
		err = ppro.presetSave(filename)
	} else {
		err = fmt.Errorf("PalettePro.Api: unhandled save category %s", category)
	}
	if err != nil {
		engine.LogError(err)
	}
	return err
}

func (ppro *PalettePro) presetSave(presetName string) error {

	category := "preset"
	path := engine.WritableFilePath(category, presetName)

	engine.LogOfType("saved", "PalettePro.presetSave", "preset", presetName)

	sortedLayerNames := []string{}
	for _, layer := range ppro.layer {
		sortedLayerNames = append(sortedLayerNames, layer.Name())
	}
	sort.Strings(sortedLayerNames)

	s := "{\n    \"params\": {\n"
	sep := ""
	for _, layerName := range sortedLayerNames {
		layer := ppro.layer[layerName]
		sortedNames := layer.ParamNames()
		for _, fullName := range sortedNames {
			valstring := layer.Get(fullName)
			s += fmt.Sprintf("%s        \"%s-%s\":\"%s\"", sep, layer.Name(), fullName, valstring)
			sep = ",\n"
		}
	}
	// ppro.params contains the misc.* params
	s += sep + ppro.miscparams.JsonValues()
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}

// func (ppro *PalettePro) saveMisc() error {
// 	return ctx.SaveParams(ppro.miscparams, "misc", "_Current")
// }

/*
	s := "{\n    \"params\": {\n"
	s += ppro.miscparams.JsonValues()
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}
*/

func (ppro *PalettePro) addLayer(ctx *engine.PluginContext, name string) *engine.Layer {
	layer := engine.NewLayer(name)
	layer.AddListener(ctx)
	ppro.layer[name] = layer
	ppro.layerLogic[name] = NewLayerLogic(ppro, layer)
	return layer
}

/*
func (ppro *PalettePro) scheduleNoteNow(ctx *engine.PluginContext, dest string, pitch, velocity uint8, duration engine.Clicks) {
	engine.LogInfo("PalettePro.scheculeNoteNow", "dest", dest, "pitch", pitch)
	pe := &engine.PhraseElement{Value: engine.NewNoteFull(0, pitch, velocity, duration)}
	phr := engine.NewPhrase().InsertElement(pe)
	phr.Destination = dest
	ctx.SchedulePhrase(phr, ctx.CurrentClick(), dest)
}
*/

/*
func (ppro *PalettePro) channelToDestination(channel int) string {
	return fmt.Sprintf("P_03_C_%02d", channel)
}
*/

/*
func (ppro *PalettePro) cursorToLayer(ce engine.CursorEvent) *engine.Layer {
	// For the moment, the cursor to layer mapping is 1-to-1.
	// I.e. ce.Source of "a" maps to layer "a"
	layer, ok := ppro.layer[ce.Source()]
	if !ok {
		return nil
	}
	return layer
}

func (ppro *PalettePro) channelToLayer(channel int) *engine.Layer {
	// For the moment, the cursor to layer mapping is 1-to-1.
	// I.e. ce.Source of "a" maps to layer "a"
	ch2lay := map[int]string{
		1: "A", 2: "B", 3: "C", 4: "D",
		5: "E", 6: "F", 7: "G", 8: "H",
	}
	layerName, ok := ch2lay[channel]
	if !ok {
		return ppro.layer["A"]
	}
	layer, ok := ppro.layer[layerName]
	if !ok {
		return ppro.layer["A"]
	}
	return layer
}
*/

// SetExternalScale xxx
func (ppro *PalettePro) setExternalScale(pitch int, on bool) {
	s := ppro.externalScale
	for p := pitch; p < 128; p += 12 {
		s.HasNote[p] = on
	}
}

func (ppro *PalettePro) handleMIDISetScaleNote(e engine.MidiEvent) {
	status := e.Status() & 0xf0
	pitch := int(e.Data1())
	if status == 0x90 {
		// If there are no notes held down (i.e. this is the first), clear the scale
		if ppro.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a noteon
			ppro.MIDINumDown = 0
		}
		if ppro.MIDINumDown == 0 {
			ppro.clearExternalScale()
		}
		ppro.setExternalScale(pitch%12, true)
		ppro.MIDINumDown++
		if pitch < 60 {
			ppro.MIDIOctaveShift = -1
		} else if pitch > 72 {
			ppro.MIDIOctaveShift = 1
		} else {
			ppro.MIDIOctaveShift = 0
		}
	} else if status == 0x80 {
		ppro.MIDINumDown--
	}
}

func (ppro *PalettePro) doAttractAction(ctx *engine.PluginContext) {

	now := time.Now()
	dt := now.Sub(ppro.lastAttractGestureTime).Seconds()
	if ppro.attractModeIsOn && dt > ppro.attractGestureInterval {

		sourceNames := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		source := sourceNames[i]
		ppro.lastAttractGestureTime = now

		cid := fmt.Sprintf("%s#%d,internal", source, ppro.nextCursorNum)
		ppro.nextCursorNum++

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		noteDuration := time.Second
		go ctx.GenerateCursorGesture(cid, noteDuration, x0, y0, z0, x1, y1, z1)
		ppro.lastAttractGestureTime = now
	}

	dp := now.Sub(ppro.lastAttractPresetChange).Seconds()
	if ppro.attractPreset == "random" && dp > ppro.attractPresetInterval {
		ppro.loadPresetRand(ctx)
		ppro.lastAttractPresetChange = now
	}
}

func (ppro *PalettePro) mmttInfo() *engine.ProcessInfo {

	// NOTE: it's inside a sub-directory of bin, so all the necessary .dll's are contained

	// The value of mmtt is either "kinect" or "oak"
	mmtt := engine.ConfigValueWithDefault("mmtt", "kinect")
	fullpath := filepath.Join(engine.PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !engine.FileExists(fullpath) {
		engine.LogWarn("no mmtt executable found, looking for", "path", fullpath)
		fullpath = ""
	}
	return engine.NewProcessInfo("mmtt_"+mmtt+".exe", fullpath, "", nil)
}

func (ppro *PalettePro) guiInfo() *engine.ProcessInfo {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(engine.PaletteDir(), "bin", "pyinstalled", exe)
	return engine.NewProcessInfo(exe, fullpath, "", nil)
}
