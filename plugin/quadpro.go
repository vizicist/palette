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

	"github.com/vizicist/palette/engine"
)

type QuadPro struct {

	// Per-patch things
	patch      map[string]*engine.Patch
	patchLogic map[string]*PatchLogic

	started bool

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
	quadpro := &QuadPro{
		patch:      map[string]*engine.Patch{},
		patchLogic: map[string]*PatchLogic{},
		// miscparams:   engine.NewParamValues(),
		// engineClicks: engine.NewParamValues(),

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
	engine.RegisterPlugin("quadpro", quadpro.Api)
}

func (quadpro *QuadPro) Api(ctx *engine.PluginContext, api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "start":
		return "", quadpro.start(ctx)

	case "stop":
		quadpro.started = false
		return "", ctx.StopRunning("all")

	case "set":
		return quadpro.onSet(ctx, apiargs)

	case "get":
		return quadpro.onGet(apiargs)

	case "event":
		eventName, ok := apiargs["event"]
		if !ok {
			return "", fmt.Errorf("QuadPro. Missing event argument")
		}
		if eventName == "click" {
			return quadpro.onClick(ctx, apiargs)
		}

		// engine.LogInfo("Ppro.event", "eventName", eventName)
		switch eventName {
		case "midi":
			return quadpro.onMidiEvent(ctx, apiargs)
		case "cursor":
			return quadpro.onCursorEvent(ctx, apiargs)
		case "clientrestart":
			return quadpro.onClientRestart(ctx, apiargs)

		case "quad_save_current":
			return "", quadpro.saveQuad("_Current")

		case "patch_set":
			return "", quadpro.onPatchSet(ctx, apiargs)

		case "patch_refresh_all":
			return "", quadpro.onPatchRefreshAll(apiargs)

		default:
			return "", fmt.Errorf("QuadPro. Unhandled event type %s", eventName)
		}

	case "ANO":
		for _, patch := range quadpro.patch {
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
		quadpro.setAttractMode(onoff)
		return "", nil

	case "clearexternalscale":
		quadpro.clearExternalScale()
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
		quadpro.setAttractMode(false)
		return "", quadpro.Load(ctx, category, filename)

	case "save":
		category, oksaved := apiargs["category"]
		if !oksaved {
			return "", fmt.Errorf("missing category parameter")
		}
		filename, oksaved := apiargs["filename"]
		if !oksaved {
			return "", fmt.Errorf("missing filename parameter")
		}
		return "", quadpro.save(category, filename)

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
			"attractmode", fmt.Sprintf("%v", quadpro.attractModeIsOn),
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
		go quadpro.doTest(ctx, ntimes, dt)
		return "", nil

	default:
		engine.LogWarn("QuadPro.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("QuadPro.Api unrecognized api=%s", api)
	}
}

func (quadpro *QuadPro) onPatchRefreshAll(apiargs map[string]string) error {
	patchName, ok := apiargs["patch"]
	if !ok {
		return fmt.Errorf("QuadPro.onSet: no patch argument")
	}
	patch := engine.GetPatch(patchName)
	if patch == nil {
		return fmt.Errorf("no such patch: %s", patchName)
	}
	patch.RefreshAllPatchValues()
	return nil
}

func (quadpro *QuadPro) start(ctx *engine.PluginContext) error {

	if quadpro.started {
		return fmt.Errorf("QuadPro. already started")
	}
	quadpro.started = true

	quadpro.clearExternalScale()

	// Make sure all the global parameters are set to their initial values
	// XXX - Is this right?  Don't they get read in from global._Current.json ?
	// for nm, pd := range engine.ParamDefs {
	// 	if pd.Category == "engine" {
	// 		err := quadpro.engineClicks.SetParamValueWithString(nm, pd.Init)
	// 		if err != nil {
	// 			engine.LogError(err)
	// 		}
	// 	}
	// }

	transposebeats := engine.Clicks(engine.ConfigIntWithDefault("transposebeats", 48))
	quadpro.transposeNext = transposebeats * engine.OneBeat
	quadpro.transposeClicks = transposebeats

	quadpro.generateVisuals = engine.ConfigBoolWithDefault("generatevisuals", true)
	quadpro.generateSound = engine.ConfigBoolWithDefault("generatesound", true)
	quadpro.transposeAuto = engine.ConfigBoolWithDefault("transposeauto", true)

	ctx.AddProcessBuiltIn("resolume")
	ctx.AddProcessBuiltIn("bidule")
	ctx.AddProcess("gui", quadpro.guiInfo())
	ctx.AddProcess("mmtt", quadpro.mmttInfo())

	ctx.AllowSource("A", "B", "C", "D")

	_ = quadpro.addPatch(ctx, "A")
	_ = quadpro.addPatch(ctx, "B")
	_ = quadpro.addPatch(ctx, "C")
	_ = quadpro.addPatch(ctx, "D")

	engine.LogError(quadpro.Load(ctx, "quad", "_Current"))

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	quadpro.processCheckSecs = engine.ConfigFloatWithDefault("processchecksecs", 60)

	quadpro.attractCheckSecs = engine.ConfigFloatWithDefault("attractchecksecs", 2)
	quadpro.attractIdleSecs = 60 * engine.ConfigFloatWithDefault("attractidleminutes", 0)

	quadpro.attractChangeInterval = engine.ConfigFloatWithDefault("attractchangeinterval", 30)
	quadpro.attractGestureInterval = engine.ConfigFloatWithDefault("attractgestureinterval", 0.5)

	return nil
}

func (quadpro *QuadPro) onCursorEvent(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {

	ddu, ok := apiargs["ddu"]
	if !ok {
		return "", fmt.Errorf("QuadPro. Missing ddu argument")
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
		quadpro.setAttractMode(false)
	}

	// For the moment, the cursor to patchLogic mapping is 1-to-1.
	// I.e. ce.Source of "A" maps to patchLogic "A"
	patchLogic, ok := quadpro.patchLogic[ce.Source()]
	if !ok || patchLogic == nil {
		return "", nil
	}
	cursorStyle := patchLogic.patch.Get("misc.cursorstyle")
	if quadpro.generateSound && !quadpro.attractModeIsOn {
		patchLogic.generateSoundFromCursor(ctx, ce, cursorStyle)
	}
	if quadpro.generateVisuals {
		patchLogic.generateVisualsFromCursor(ce)
	}
	return "", nil
}

func (quadpro *QuadPro) setAttractMode(onoff bool) {

	if onoff == quadpro.attractModeIsOn {
		return // already in that mode
	}
	// Throttle it a bit
	secondsSince := time.Since(quadpro.lastAttractModeChange).Seconds()
	if secondsSince > 1.0 {
		engine.LogOfType("attract", "QuadPro. changing attract", "onoff", onoff)
		quadpro.attractModeIsOn = onoff
		quadpro.lastAttractModeChange = time.Now()
	}
}

func (quadpro *QuadPro) clearExternalScale() {
	quadpro.externalScale = engine.MakeScale()
}

func (quadpro *QuadPro) onClientRestart(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {
	// These are messages from the Palette FFGL plugins in Resolume,
	// telling us that they have restarted, and we should resend all parameters
	portnum, ok := apiargs["portnum"]
	if !ok {
		return "", fmt.Errorf("QuadPro. Missing portnum argument")
	}
	ffglportnum, err := strconv.Atoi(portnum)
	if err != nil {
		return "", err
	}
	engine.LogInfo("quadpro got clientrestart", "portnum", portnum)
	// The restart message contains the
	// portnum of the plugin that restarted.
	for _, patch := range quadpro.patch {
		e := patch.RefreshAllIfPortnumMatches(ffglportnum)
		if e != nil {
			engine.LogError(e)
			err = e
		}
	}
	return "", err
}

func (quadpro *QuadPro) onPatchSet(ctx *engine.PluginContext, apiargs map[string]string) error {
	paramName, paramValue, err := engine.GetNameValue(apiargs)
	if err != nil {
		return fmt.Errorf("QuadPro.onSet: %s", err)
	}
	patchName, ok := apiargs["patch"]
	if !ok {
		return fmt.Errorf("QuadPro.onSet: no patch argument")
	}
	patch := engine.GetPatch(patchName)
	if patch == nil {
		return fmt.Errorf("no such patch: %s", patchName)
	}
	return patch.Set(paramName, paramValue)
}

// onSet is only used for global parameters.
func (quadpro *QuadPro) onSet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {

	paramName, paramValue, err := engine.GetNameValue(apiargs)
	if err != nil {
		return "", fmt.Errorf("QuadPro.onSet: %s", err)
	}

	if strings.HasPrefix(paramName, "engine") {
		return "", engine.TheEngine.Set(paramName, paramValue)
	}
	return "", fmt.Errorf("QuadPro.onSet: can't handle non-global parameter %s", paramName)
}

func (quadpro *QuadPro) onGet(apiargs map[string]string) (result string, err error) {
	paramName, ok := apiargs["name"]
	if !ok {
		return "", fmt.Errorf("QuadPro.onPatchGet: Missing name argument")
	}
	if strings.HasPrefix(paramName, "engine") {
		return engine.TheEngine.Get(paramName), nil
	} else {
		return "", fmt.Errorf("QuadPro.onGet: can't handle parameter %s", paramName)
	}
}

func (quadpro *QuadPro) onClick(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {
	quadpro.checkAttract(ctx)
	quadpro.checkProcess(ctx)
	return "", nil
}

func (quadpro *QuadPro) checkProcess(ctx *engine.PluginContext) {

	firstTime := (quadpro.lastProcessCheck == time.Time{})

	// If processCheckSecs is 0, process checking is disabled
	processCheckEnabled := quadpro.processCheckSecs > 0
	if !processCheckEnabled {
		if firstTime {
			engine.LogInfo("Process Checking is disabled.")
		}
		return
	}

	now := time.Now()
	sinceLastProcessCheck := now.Sub(quadpro.lastProcessCheck).Seconds()
	if processCheckEnabled && (firstTime || sinceLastProcessCheck > quadpro.processCheckSecs) {
		// Put it in background, so calling
		// tasklist or ps doesn't disrupt realtime
		// The sleep here is because if you kill something and
		// immediately check to see if it's running, it reports that
		// it's stil running.
		go func() {
			time.Sleep(2 * time.Second)
			ctx.CheckAutostartProcesses()
		}()
		quadpro.lastProcessCheck = now
	}
}

func (quadpro *QuadPro) checkAttract(ctx *engine.PluginContext) {

	// Every so often we check to see if attract mode should be turned on
	now := time.Now()
	attractModeEnabled := quadpro.attractIdleSecs > 0
	sinceLastAttractCheck := now.Sub(quadpro.lastAttractCheck).Seconds()
	if attractModeEnabled && sinceLastAttractCheck > quadpro.attractCheckSecs {
		quadpro.lastAttractCheck = now
		// There's a delay when checking cursor activity to turn attract mod on.
		// Non-internal cursor activity turns attract mode off instantly.
		sinceLastAttractModeChange := time.Since(quadpro.lastAttractModeChange).Seconds()
		if !quadpro.attractModeIsOn && sinceLastAttractModeChange > quadpro.attractIdleSecs {
			// Nothing happening for a while, turn attract mode on
			quadpro.setAttractMode(true)
		}
	}

	if quadpro.attractModeIsOn {
		quadpro.doAttractAction(ctx)
	}
}

func (quadpro *QuadPro) onMidiEvent(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {

	me, err := engine.MidiEventFromMap(apiargs)
	if err != nil {
		return "", err
	}

	engine.LogInfo("QuadPro.onMidiEvent", "me", me)

	/*
		if EraeEnabled {
			HandleEraeMIDI(me)
		}
	*/

	if quadpro.MIDIThru {
		engine.LogInfo("PassThruMIDI", "msg", me.Msg)
		// ctx.ScheduleMidi(patch, me.Msg, ctx.CurrentClick())
		ctx.ScheduleAt(me.Msg, ctx.CurrentClick())
	}
	if quadpro.MIDISetScale {
		quadpro.handleMIDISetScaleNote(me)
	}

	return "", nil
}

func (quadpro *QuadPro) doTest(ctx *engine.PluginContext, ntimes int, dt time.Duration) {
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
		_, err := quadpro.Api(ctx, "event", thismap)
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
		_, err = quadpro.Api(ctx, "event", thismap)
		if err != nil {
			engine.LogError(err)
			return
		}
	}
	engine.LogInfo("doTest end")
}

func (quadpro *QuadPro) loadQuadRand(ctx *engine.PluginContext) error {

	arr, err := engine.SavedFileList("quad")
	if err != nil {
		return err
	}
	rn := rand.Uint64() % uint64(len(arr))
	engine.LogInfo("loadQuadRand", "quad", arr[rn])

	engine.LogError(quadpro.Load(ctx, "quad", arr[rn]))

	for _, patch := range quadpro.patch {
		patch.RefreshAllPatchValues()
	}
	return nil
}

func (quadpro *QuadPro) Load(ctx *engine.PluginContext, category string, filename string) error {

	path := engine.SavedFilePath(category, filename)
	paramsMap, err := engine.LoadParamsMap(path)
	if err != nil {
		engine.LogError(err)
		return err
	}

	engine.LogInfo("QuadPro.Load", "category", category, "filename", filename)

	var lasterr error

	switch category {

	case "quad":
		for _, patch := range quadpro.patch {
			err := patch.ApplyPatchValuesFromQuadMap(paramsMap)
			if err != nil {
				engine.LogError(err)
				lasterr = err
			}
			patch.RefreshAllPatchValues()
		}

	default:
		engine.LogWarn("QuadPro.Load: unhandled", "category", category, "filename", filename)
	}

	// Decide what _Current things we should save
	// when we load something.  E.g. if we're
	// loading a patch with something, we want to
	// then save the entire quad
	err = nil
	switch category {
	case "engine":
		if filename != "_Current" {
			// No need to save _Current if we're loading it
			err = quadpro.save("engine", "_Current")
		}
	case "quad":
		if filename != "_Current" {
			err = quadpro.saveQuad("_Current")
		}
	case "patch", "sound", "visual", "effect", "misc":
		// If we're loading a patch (or something inside a patch, like sound, visual, etc),
		// we save the entire quad, since that's our real persistent state
		if filename != "_Current" {
			err = quadpro.saveQuad("_Current")
		}
	}
	if err != nil {
		engine.LogError(err)
		lasterr = err
	}
	return lasterr
}

func (quadpro *QuadPro) save(category string, filename string) (err error) {

	engine.LogOfType("saved", "QuadPro.save", "category", category, "filename", filename)

	if category == "engine" {
		// err = quadpro.engineClicks.Save("engine", filename)
		engine.LogWarn("QuadPro.save: shouldn't be saving global?")
		err = fmt.Errorf("QuadPro.save: global shouldn't be handled here")
	} else if category == "quad" {
		err = quadpro.saveQuad(filename)
	} else {
		err = fmt.Errorf("QuadPro.Api: unhandled save category %s", category)
	}
	if err != nil {
		engine.LogError(err)
	}
	return err
}

func (quadpro *QuadPro) saveQuad(quadName string) error {

	category := "quad"
	path := engine.WritableFilePath(category, quadName)

	engine.LogOfType("saved", "QuadPro.saveQuad", "quad", quadName)

	sortedPatchNames := []string{}
	for _, patch := range quadpro.patch {
		sortedPatchNames = append(sortedPatchNames, patch.Name())
	}
	sort.Strings(sortedPatchNames)

	s := "{\n    \"params\": {\n"
	sep := ""
	for _, patchName := range sortedPatchNames {
		patch := quadpro.patch[patchName]
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

func (quadpro *QuadPro) addPatch(ctx *engine.PluginContext, name string) *engine.Patch {
	patch := engine.NewPatch(name)
	patch.AddListener(ctx)
	quadpro.patch[name] = patch
	quadpro.patchLogic[name] = NewPatchLogic(quadpro, patch)
	return patch
}

/*
func (quadpro *QuadPro) scheduleNoteNow(ctx *engine.PluginContext, dest string, pitch, velocity uint8, duration engine.Clicks) {
	engine.LogInfo("QuadPro.scheculeNoteNow", "dest", dest, "pitch", pitch)
	pe := &engine.PhraseElement{Value: engine.NewNoteFull(0, pitch, velocity, duration)}
	phr := engine.NewPhrase().InsertElement(pe)
	phr.Destination = dest
	ctx.SchedulePhrase(phr, ctx.CurrentClick(), dest)
}
*/

// SetExternalScale xxx
func (quadpro *QuadPro) setExternalScale(pitch int, on bool) {
	s := quadpro.externalScale
	for p := pitch; p < 128; p += 12 {
		s.HasNote[p] = on
	}
}

func (quadpro *QuadPro) handleMIDISetScaleNote(e engine.MidiEvent) {
	status := e.Status() & 0xf0
	pitch := int(e.Data1())
	if status == 0x90 {
		// If there are no notes held down (i.e. this is the first), clear the scale
		if quadpro.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a noteon
			quadpro.MIDINumDown = 0
		}
		if quadpro.MIDINumDown == 0 {
			quadpro.clearExternalScale()
		}
		quadpro.setExternalScale(pitch%12, true)
		quadpro.MIDINumDown++
		if pitch < 60 {
			quadpro.MIDIOctaveShift = -1
		} else if pitch > 72 {
			quadpro.MIDIOctaveShift = 1
		} else {
			quadpro.MIDIOctaveShift = 0
		}
	} else if status == 0x80 {
		quadpro.MIDINumDown--
	}
}

func (quadpro *QuadPro) doAttractAction(ctx *engine.PluginContext) {

	now := time.Now()
	dt := now.Sub(quadpro.lastAttractGestureTime).Seconds()
	if quadpro.attractModeIsOn && dt > quadpro.attractGestureInterval {

		sourceNames := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		source := sourceNames[i]
		quadpro.lastAttractGestureTime = now

		cid := fmt.Sprintf("%s#%d,internal", source, quadpro.nextCursorNum)
		quadpro.nextCursorNum++

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		noteDuration := time.Second
		go ctx.GenerateCursorGesture(cid, noteDuration, x0, y0, z0, x1, y1, z1)
		quadpro.lastAttractGestureTime = now
	}

	dp := now.Sub(quadpro.lastAttractChange).Seconds()
	if dp > quadpro.attractChangeInterval {
		err := quadpro.loadQuadRand(ctx)
		if err != nil {
			engine.LogError(err)
		}
		quadpro.lastAttractChange = now
	}
}

func (quadpro *QuadPro) mmttInfo() *engine.ProcessInfo {

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

func (quadpro *QuadPro) guiInfo() *engine.ProcessInfo {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(engine.PaletteDir(), "bin", "pyinstalled", exe)
	return engine.NewProcessInfo(exe, fullpath, "", nil)
}
