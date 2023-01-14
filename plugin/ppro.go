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

	// Per-patch things
	patch      map[string]*engine.Patch
	patchLogic map[string]*PatchLogic

	// PalettePro parameters
	// miscparams   *engine.ParamValues
	globalparams *engine.ParamValues

	started        bool
	// midiinputpatch *engine.Patch

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

	attractModeIsOn        bool
	lastAttractModeChange  time.Time
	lastAttractGestureTime time.Time
	lastAttractChange      time.Time

	// parameters
	attractGestureInterval float64
	attractChangeInterval  float64

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
		patch:      map[string]*engine.Patch{},
		patchLogic: map[string]*PatchLogic{},
		// miscparams:   engine.NewParamValues(),
		globalparams: engine.NewParamValues(),

		attractModeIsOn: false,
		generateVisuals: true,
		generateSound:   true,

		lastAttractModeChange:  time.Now(),
		lastAttractGestureTime: time.Now(),
		lastAttractChange:      time.Now(),
		lastAttractCheck:       time.Now(),
		lastProcessCheck:       time.Time{},

		attractGestureInterval: 0,
		attractChangeInterval:  0,
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
		case "notification_of_patch_set":
			return ppro.onPatchSet(ctx, apiargs)
		case "notification_of_patch_refresh_all":
			return ppro.onPatchRefreshAll(ctx, apiargs)
		case "alertonsave":
			// This events from a listener will trigger saving of the entire Quad
			return "", ppro.saveQuad("_Current")
		default:
			return "", fmt.Errorf("PalettePro: Unhandled event type %s", eventName)
		}

	case "ANO":
		for _, patch := range ppro.patch {
			patch.Synth.SendANO()
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

	_ = ppro.addPatch(ctx, "A")
	_ = ppro.addPatch(ctx, "B")
	_ = ppro.addPatch(ctx, "C")
	_ = ppro.addPatch(ctx, "D")

	engine.LogError(ppro.Load(ctx, "global", "_Current"))
	engine.LogError(ppro.Load(ctx, "quad", "_Current"))

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	ppro.processCheckSecs = engine.ConfigFloatWithDefault("processchecksecs", 60)

	ppro.attractCheckSecs = engine.ConfigFloatWithDefault("attractchecksecs", 2)
	ppro.attractIdleSecs = engine.ConfigFloatWithDefault("attractidlesecs", 0)

	ppro.attractChangeInterval = engine.ConfigFloatWithDefault("attractchangeinterval", 30)
	ppro.attractGestureInterval = engine.ConfigFloatWithDefault("attractgestureinterval", 0.5)

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

	// For the moment, the cursor to patchLogic mapping is 1-to-1.
	// I.e. ce.Source of "A" maps to patchLogic "A"
	patchLogic, ok := ppro.patchLogic[ce.Source()]
	if !ok || patchLogic == nil {
		return "", nil
	}
	cursorStyle := ppro.globalparams.Get("misc.cursorstyle")
	if ppro.generateSound && !ppro.attractModeIsOn {
		patchLogic.generateSoundFromCursor(ctx, ce, cursorStyle)
	}
	if ppro.generateVisuals {
		patchLogic.generateVisualsFromCursor(ce)
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
	for _, patch := range ppro.patch {
		patch.RefreshAllIfPortnumMatches(ffglportnum)
	}
	return "", nil
}

// onSet is only used for parameters that are not in patches.
// NOTE: this routine does not automatically persist the values to disk.
func (ppro *PalettePro) onSet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {

	paramName, paramValue, err := engine.GetNameValue(apiargs)
	if err != nil {
		return "", fmt.Errorf("PalettePro.onSet: %s", err)
	}

	switch {
	case strings.HasPrefix(paramName, "global"):
		return "", ppro.globalparams.Set(paramName, paramValue)

	default:
		return "", fmt.Errorf("PalettePro.onSet: can't handle parameter %s", paramName)
	}
}

func (ppro *PalettePro) onGet(apiargs map[string]string) (result string, err error) {
	paramName, ok := apiargs["name"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onPatchSet: Missing name argument")
	}
	if strings.HasPrefix(paramName, "global") {
		return ppro.globalparams.Get(paramName), nil
	} else {
		return "", fmt.Errorf("PalettePro.onGet: can't handle parameter %s", paramName)
	}
}

func (ppro *PalettePro) onPatchRefreshAll(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {
	patchName, ok := apiargs["patch"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onPatchRefreshAll: Missing patch argument")
	}
	patch, ok := ppro.patch[patchName]
	if !ok {
		return "", fmt.Errorf("PalettePro.onPatchRefreshAll: No patch named %s", patchName)
	}
	ppro.refreshAllPatchValues(ctx, patch)
	return "", nil
}

func (ppro *PalettePro) refreshAllPatchValues(ctx *engine.PluginContext, patch *engine.Patch) {
	for _, paramName := range patch.ParamNames() {
		paramValue := patch.Get(paramName)
		ppro.refreshPatchValue(ctx, patch, paramName, paramValue)
	}
}

func (ppro *PalettePro) onPatchSet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {
	patchName, ok := apiargs["patch"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onPatchSet: Missing patch argument")
	}
	paramName, paramValue, err := engine.GetNameValue(apiargs)
	if err != nil {
		return "", fmt.Errorf("PalettePro.onPatchSet: err=%s", err)
	}
	patch, ok := ppro.patch[patchName]
	if !ok {
		return "", fmt.Errorf("PalettePro.onPatchSet: No patch named %s", patchName)
	}
	ppro.refreshPatchValue(ctx, patch, paramName, paramValue)
	return "", nil
}

func (ppro *PalettePro) refreshPatchValue(ctx *engine.PluginContext, patch *engine.Patch, paramName string, paramValue string) {

	if strings.HasPrefix(paramName, "visual.") {
		name := strings.TrimPrefix(paramName, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, paramValue)
		msg.Append(args)
		engine.TheResolume().ToFreeFramePlugin(patch.Name(), msg)
	}

	if strings.HasPrefix(paramName, "effect.") {
		name := strings.TrimPrefix(paramName, "effect.")
		// Effect parameters get sent to Resolume
		engine.TheResolume().SendEffectParam(patch.Name(), name, paramValue)
	}

	if paramName == "sound.synth" {
		synth := ctx.GetSynth(paramValue)
		if synth == nil {
			engine.LogWarn("PalettePro: no synth named", "synth", paramValue)
		}
		patch.Synth = synth
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

	if ppro.MIDIThru {
		engine.LogInfo("PassThruMIDI", "msg", me.Msg)
		// ctx.ScheduleMidi(patch, me.Msg, ctx.CurrentClick())
		ctx.ScheduleAt(me.Msg, ctx.CurrentClick())
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

func (ppro *PalettePro) loadQuadRand(ctx *engine.PluginContext) {

	arr, err := engine.SavedFileList("quad")
	if err != nil {
		engine.LogError(err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	engine.LogInfo("loadQuadRand", "quad", arr[rn])

	engine.LogError(ppro.Load(ctx, "quad", arr[rn]))

	for _, patch := range ppro.patch {
		patch.AlertListenersToRefreshAll()
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

	case "quad":
		for _, patch := range ppro.patch {
			err := patch.ApplyQuadValuesFrom(paramsMap)
			if err != nil {
				engine.LogError(err)
				lasterr = err
			}
			ppro.refreshAllPatchValues(ctx, patch)
		}

	default:
		engine.LogWarn("PalettePro.Load: unhandled", "category", category, "filename", filename)
	}

	// Decide what _Current things we should save
	// when we load something.  E.g. if we're
	// loading a patch with something, we want to
	// then save the entire quad
	err = nil
	switch category {
	case "global":
		if filename == "_Current" {
			err = ppro.save("global", "_Current")
		}
	case "patch", "sound", "visual", "effect", "misc":
		if filename != "_Current" {
			err = ppro.saveQuad("_Current")
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

	if category == "global" {
		err = ppro.globalparams.Save("global", filename)
	} else if category == "quad" {
		err = ppro.saveQuad(filename)
	} else {
		err = fmt.Errorf("PalettePro.Api: unhandled save category %s", category)
	}
	if err != nil {
		engine.LogError(err)
	}
	return err
}

func (ppro *PalettePro) saveQuad(quadName string) error {

	category := "quad"
	path := engine.WritableFilePath(category, quadName)

	engine.LogOfType("saved", "PalettePro.saveQuad", "quad", quadName)

	sortedPatchNames := []string{}
	for _, patch := range ppro.patch {
		sortedPatchNames = append(sortedPatchNames, patch.Name())
	}
	sort.Strings(sortedPatchNames)

	s := "{\n    \"params\": {\n"
	sep := ""
	for _, patchName := range sortedPatchNames {
		patch := ppro.patch[patchName]
		sortedNames := patch.ParamNames()
		for _, fullName := range sortedNames {
			valstring := patch.Get(fullName)
			s += fmt.Sprintf("%s        \"%s-%s\":\"%s\"", sep, patch.Name(), fullName, valstring)
			sep = ",\n"
		}
	}
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}

func (ppro *PalettePro) addPatch(ctx *engine.PluginContext, name string) *engine.Patch {
	patch := engine.NewPatch(name)
	patch.AddListener(ctx)
	ppro.patch[name] = patch
	ppro.patchLogic[name] = NewPatchLogic(ppro, patch)
	return patch
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

	dp := now.Sub(ppro.lastAttractChange).Seconds()
	if dp > ppro.attractChangeInterval {
		ppro.loadQuadRand(ctx)
		ppro.lastAttractChange = now
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
