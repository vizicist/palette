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

var AliveOutputPort = 3331

type PalettePro struct {

	// Per-layer things
	layer map[string]*engine.Layer
	logic map[string]*LayerLogic

	// PalettePro parameters
	params *engine.ParamValues

	started        bool
	resolume       *Resolume
	bidule         *Bidule
	processManager *ProcessManager
	midiinputlayer *engine.Layer

	generateVisuals bool
	generateSound   bool

	MIDIOctaveShift  int
	MIDINumDown      int
	MIDIThru         bool
	MIDIThruScadjust bool
	MIDISetScale     bool
	MIDIQuantized    bool
	MIDIUseScale     bool // if true, scadjust uses "external" Scale
	TransposePitch   int
	externalScale    *engine.Scale

	attractModeIsOn        bool
	lastAttractCommand     time.Time
	lastAttractGestureTime time.Time
	lastAttractRandTime    time.Time
	attractGestureDuration time.Duration
	attractNoteDuration    time.Duration
	attractPresetDuration  time.Duration
	attractPreset          string
	attractClient          *osc.Client
	lastAttractChange      float64
	lastAttractCheck       float64
	attractCheckSecs       float64
	attractIdleSecs        float64
	aliveSecs              float64
	lastAlive              float64
	lastProcessCheck       float64
	processCheckSecs       float64
	scale                  *engine.Scale

	transposeAuto   bool
	transposeNext   engine.Clicks
	transposeClicks engine.Clicks // time between auto transpose changes
	transposeIndex  int           // current place in tranposeValues
	transposeValues []int
}

func init() {
	transposebeats := engine.Clicks(engine.ConfigIntWithDefault("transposebeats", 48))
	ppro := &PalettePro{
		layer:  map[string]*engine.Layer{},
		logic:  map[string]*LayerLogic{},
		params: engine.NewParamValues(),

		attractModeIsOn: false,
		generateVisuals: engine.ConfigBoolWithDefault("generatevisuals", true),
		generateSound:   engine.ConfigBoolWithDefault("generatesound", true),

		lastAttractCommand:     time.Time{},
		lastAttractGestureTime: time.Time{},
		lastAttractRandTime:    time.Time{},
		attractGestureDuration: 0,
		attractNoteDuration:    0,
		attractPresetDuration:  0,
		attractPreset:          "",
		attractClient:          &osc.Client{},
		lastAttractChange:      0,
		attractCheckSecs:       0,
		lastAttractCheck:       0,
		attractIdleSecs:        0,
		aliveSecs:              float64(engine.ConfigFloatWithDefault("alivesecs", 5)),
		lastAlive:              0,
		lastProcessCheck:       0,
		processCheckSecs:       0,
		scale:                  engine.GetScale("newage"),

		transposeAuto:   engine.ConfigBoolWithDefault("transposeauto", true),
		transposeNext:   transposebeats * engine.OneBeat,
		transposeClicks: transposebeats,
		transposeIndex:  0,
		transposeValues: []int{0, -2, 3, -5},
	}
	ppro.clearExternalScale()
	engine.RegisterPlugin("ppro", ppro.Api)
}

func (ppro *PalettePro) Api(ctx *engine.PluginContext, api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "start":
		return "", ppro.start(ctx)

	case "stop":
		ppro.started = false
		return "", ppro.processManager.StopRunning(ctx, "all")

	case "set":
		return ppro.onSet(ctx, apiargs)

	case "get":
		return ppro.onGet(ctx, apiargs)

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
		case "layerset":
			return ppro.onLayerSet(ctx, apiargs)
		default:
			return "", fmt.Errorf("PalettePro: Unhandled event type %s", eventName)
		}

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
		filename, oksaved := apiargs["filename"]
		if !oksaved {
			return "", fmt.Errorf("missing filename parameter")
		}

		err := ppro.loadPreset(filename)
		if err != nil {
			return "", err
		}
		err = nil
		for _, layer := range ppro.layer {
			e := layer.SaveCurrentParams()
			if e != nil {
				err = e
			}
		}
		return "", err

	case "save":
		filename, oksaved := apiargs["filename"]
		if !oksaved {
			return "", fmt.Errorf("missing filename parameter")
		}
		return "", ppro.savePreset(filename)

	case "startprocess":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("ExecuteAPI: missing process argument")
		} else {
			err = ppro.processManager.StartRunning(ctx, process)
		}
		return "", err

	case "stopprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("ExecuteAPI: missing process argument")
		}
		return "", ppro.processManager.StopRunning(ctx, process)

	case "activate":
		// Force Activate even if already activated
		ppro.processManager.ActivateAll()
		return "", nil

	case "killall":
		return "", ppro.processManager.killAll(ctx)

	case "nextalive":
		// acts like a timer, but it could wait for
		// some event if necessary
		time.Sleep(1 * time.Second)
		result = engine.JsonObject(
			"event", "alive",
			"seconds", fmt.Sprintf("%f", ctx.Uptime()),
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
		engine.LogWarn("Pro.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecuteSavedAPI unrecognized api=%s", api)
	}
}

func (ppro *PalettePro) start(ctx *engine.PluginContext) error {

	if ppro.started {
		return fmt.Errorf("PalettePro: already started")
	}
	ppro.started = true
	ppro.resolume = NewResolume()
	ppro.bidule = NewBidule()

	ppro.processManager = NewProcessManager()

	ppro.processManager.AddProcess("resolume", ppro.resolume.ProcessInfo(ctx))
	ppro.processManager.AddProcess("bidule", ppro.bidule.ProcessInfo())
	ppro.processManager.AddProcess("gui", ppro.guiInfo())
	ppro.processManager.AddProcess("mmtt", ppro.mmttInfo())

	ctx.AllowSource("A", "B", "C", "D")

	layerA := ppro.addLayer(ctx, "A")
	layerA.Set("visual.shape", "circle")
	layerA.LoadSaved("layer", "White_Ghosts")

	layerB := ppro.addLayer(ctx, "B")
	layerB.Set("visual.shape", "square")
	layerB.LoadSaved("layer", "Concentric_Squares")

	layerC := ppro.addLayer(ctx, "C")
	layerC.Set("visual.shape", "square")
	layerC.LoadSaved("layer", "Circular_Moire")

	layerD := ppro.addLayer(ctx, "D")
	layerD.Set("visual.shape", "square")
	layerD.LoadSaved("layer", "Diagonal_Mirror")

	//ctx.ApplySaved("saved.Quick Scat_Circles")

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	ppro.processCheckSecs = float64(engine.ConfigFloatWithDefault("processchecksecs", 60))

	ppro.attractCheckSecs = float64(engine.ConfigFloatWithDefault("attractchecksecs", 2))
	ppro.attractIdleSecs = float64(engine.ConfigFloatWithDefault("attractidlesecs", 0))

	secs1 := engine.ConfigFloatWithDefault("attractpresetduration", 30)
	ppro.attractPresetDuration = time.Duration(int(secs1 * float32(time.Second)))

	secs := engine.ConfigFloatWithDefault("attractgestureduration", 0.5)
	ppro.attractGestureDuration = time.Duration(int(secs * float32(time.Second)))

	secs = engine.ConfigFloatWithDefault("attractnoteduration", 0.2)
	ppro.attractNoteDuration = time.Duration(int(secs * float32(time.Second)))

	ppro.attractPreset = engine.ConfigStringWithDefault("attractpreset", "random")

	return nil
}

func (ppro *PalettePro) clearExternalScale() {
	ppro.externalScale = engine.MakeScale()
}

func (ppro *PalettePro) onClientRestart(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {
	// These are messages from the Palette FFGL plugins in Resolume,
	// telling us that they have restarted, and we should resend all parameters
	apiportnum, ok := apiargs["portnum"]
	if !ok {
		return "", fmt.Errorf("PalettePro: Missing portnum argument")
	}
	apipnum, err := strconv.Atoi(apiportnum)
	if err != nil {
		return "", err
	}
	engine.LogInfo("ppro got clientrestart", "apipnum", apipnum)
	for nm, layer := range ppro.layer {
		portnum, _ := ppro.resolume.PortAndLayerNumForLayer(nm)
		if portnum == apipnum {
			layer.ResendAllParameters()
		}
	}
	return "", nil
}

func (ppro *PalettePro) onSet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {
	paramName, ok := apiargs["name"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onSet: Missing name argument")
	}
	paramValue, ok := apiargs["value"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onSet: Missing value argument")
	}
	ppro.params.Set(paramName, paramValue)
	engine.LogInfo("PalettePro.onSet", "name", paramName, "value", paramValue)
	return "", nil
}

func (ppro *PalettePro) onGet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {
	paramName, ok := apiargs["name"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerSet: Missing name argument")
	}
	paramValue := ppro.params.Get(paramName)
	engine.LogInfo("PalettePro.onGet", "name", paramName, "value", paramValue)
	return paramValue, nil
}

func (ppro *PalettePro) onLayerSet(ctx *engine.PluginContext, apiargs map[string]string) (result string, err error) {
	layerName, ok := apiargs["layer"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerSet: Missing layer argument")
	}
	paramName, ok := apiargs["name"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerSet: Missing name argument")
	}
	paramValue, ok := apiargs["value"]
	if !ok {
		return "", fmt.Errorf("PalettePro.onLayerSet: Missing value argument")
	}
	layer, ok := ppro.layer[layerName]
	if !ok {
		engine.LogWarn("PalettePro.onLayerSet: No layer named", "layer", layerName)
		return
	}

	if paramName == "misc.midiinputlayer" {
		ppro.midiinputlayer = ctx.GetLayer(layerName)
	}

	if strings.HasPrefix(paramName, "visual.") {
		name := strings.TrimPrefix(paramName, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, paramValue)
		if name == "ffglport" {
			engine.LogInfo("ffglport seen")
		}
		msg.Append(args)
		ppro.resolume.toFreeFramePlugin(layer.Name(), msg)
	}

	if strings.HasPrefix(paramName, "effect.") {
		name := strings.TrimPrefix(paramName, "effect.")
		// Effect parameters get sent to Resolume
		ppro.resolume.sendEffectParam(layer.Name(), name, paramValue)
	}

	if paramName == "sound.synth" {
		synth := ctx.GetSynth(paramValue)
		if synth == nil {
			engine.LogWarn("PalettePro: no synth named", "synth", paramValue)
		}
		ppro.layer[layerName].Synth = synth
	}
	return "", nil
}

func (ppro *PalettePro) onClick(ctx *engine.PluginContext, apiargs map[string]string) (string, error) {

	uptimesecs := ctx.Uptime()
	// Every so often we check to see if attract mode should be turned on
	attractModeEnabled := ppro.attractIdleSecs > 0
	sinceLastAttractChange := uptimesecs - ppro.lastAttractChange
	sinceLastAttractCheck := uptimesecs - ppro.lastAttractCheck
	if attractModeEnabled && sinceLastAttractCheck > ppro.attractCheckSecs {
		ppro.lastAttractCheck = uptimesecs
		// There's a delay when checking cursor activity to turn attract mod on.
		// Non-internal cursor activity turns attract mode off instantly.
		if !ppro.attractModeIsOn && sinceLastAttractChange > ppro.attractIdleSecs {
			// Nothing happening for a while, turn attract mode on
			engine.LogWarn("PalettePro.OnClick: should be turning attractmode on")
			ppro.lastAttractCommand = time.Now()
		}
	}

	if ppro.attractModeIsOn {
		ppro.doAttractAction(ctx)
	}

	sinceLastAlive := uptimesecs - ppro.lastAlive
	if sinceLastAlive > ppro.aliveSecs {
		ppro.publishOscAlive(ctx, uptimesecs)
		ppro.lastAlive = uptimesecs
	}

	processCheckEnabled := ppro.processCheckSecs > 0

	// At the beginning and then every processCheckSecs seconds
	// we check to see if necessary processes are still running
	firstTime := (ppro.lastProcessCheck == 0)
	sinceLastProcessCheck := uptimesecs - ppro.lastProcessCheck
	if processCheckEnabled && (firstTime || sinceLastProcessCheck > ppro.processCheckSecs) {
		// Put it in background, so calling
		// tasklist or ps doesn't disrupt realtime
		// The sleep here is because if you kill something and
		// immediately check to see if it's running, it reports that
		// it's stil running.
		go func() {
			engine.DebugLogOfType("scheduler", "Scheduler: checking processes")
			time.Sleep(2 * time.Second)
			ppro.processManager.checkAutostartProcesses(ctx)
		}()
		ppro.lastProcessCheck = uptimesecs
	}
	if !processCheckEnabled && firstTime {
		engine.LogInfo("Process Checking is disabled.")
	}
	return "", nil
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
	layerName := ppro.params.GetStringValue("misc.midiinputlayer", "A")
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
	// func (ppro *PalettePro) Api(ctx *engine.PluginContext, api string, apiargs map[string]string) (result string, err error) {
	engine.LogInfo("doTest start", "ntimes", ntimes, "dt", dt)
	for n := 0; n < ntimes; n++ {
		if n > 0 {
			time.Sleep(dt)
		}
		source := string("ABCD"[rand.Int()%4])
		thismap := map[string]string{
			"source": source,
			"event":  "cursor",
			"ddu":    "down",
			"x":      fmt.Sprintf("%f", rand.Float32()),
			"y":      fmt.Sprintf("%f", rand.Float32()),
			"z":      fmt.Sprintf("%f", rand.Float32()),
		}
		_, err := ppro.Api(ctx, "event", thismap)
		if err != nil {
			engine.LogError(err)
			return
		}
		time.Sleep(dt)
		thismap = map[string]string{
			"source": source,
			"event":  "cursor",
			"ddu":    "up",
			"x":      fmt.Sprintf("%f", rand.Float32()),
			"y":      fmt.Sprintf("%f", rand.Float32()),
			"z":      fmt.Sprintf("%f", rand.Float32()),
		}
		_, err = ppro.Api(ctx, "event", thismap)
		if err != nil {
			engine.LogError(err)
			return
		}
	}
	engine.LogInfo("doTest end")
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

	source, ok := apiargs["source"]
	if !ok {
		source = ""
	}

	id, ok := apiargs["id"]
	if !ok {
		id = ""
	}

	ce := engine.CursorEvent{
		ID:     id,
		Source: source,
		Ddu:    ddu,
		X:      x,
		Y:      y,
		Z:      z,
	}

	// Any non-internal cursor will turn attract mode off.
	if source != "internal" {
		if time.Since(ppro.lastAttractCommand) > time.Second {
			engine.LogInfo("PalettePro: shouold be turning attract mode OFF")
			ppro.lastAttractCommand = time.Now()
		}

	}

	logic := ppro.cursorToLogic(ce)
	if logic != nil {
		if ppro.generateSound {
			// XXX - HACK
			if ce.Ddu != "drag" {
				logic.generateSoundFromCursor(ctx, ce)
			}
		}
		if ppro.generateVisuals {
			logic.generateVisualsFromCursor(ce)
		}
	}

	/*
		layer := ppro.cursorToLayer(ce)
		if layer == nil {
			return "", fmt.Errorf("PalettePro: No layer for cursor ce=%+v", ce)
		}
		msg := osc.NewMessage("/sprite")
		msg.Append(ce.X)
		msg.Append(ce.Y)
		msg.Append(ce.Z)
		msg.Append(ce.ID)
		ppro.resolume.toFreeFramePlugin(layer.Name(), msg)
	*/

	return "", nil
}

func (ppro *PalettePro) loadPresetRand() {

	arr, err := engine.SavedArray("preset")
	if err != nil {
		engine.LogError(err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	engine.LogInfo("loadPresetRand", "saved", arr[rn])
	ppro.loadPreset(arr[rn])
	if err != nil {
		engine.LogError(err)
	}
}

func (ppro *PalettePro) loadPreset(presetName string) (err error) {

	path := engine.SavedFilePath("preset", presetName)
	paramsMap, err := engine.LoadParamsMap(path)
	if err != nil {
		return err
	}
	for _, layer := range ppro.layer {
		e := layer.ApplyPresetSavedMap(paramsMap)
		if e != nil {
			engine.LogError(e)
			err = e
		}
	}
	return err
}

func (ppro *PalettePro) savePreset(presetName string) error {

	// wantCategory is sound, visual, effect, snap, or quad
	category := "preset"
	path := engine.WritableFilePath(category, presetName)
	s := "{\n    \"params\": {\n"

	engine.LogInfo("savePreset", "preset", presetName)

	sortedLayerNames := []string{}
	for _, layer := range ppro.layer {
		sortedLayerNames = append(sortedLayerNames, layer.Name())
	}
	sort.Strings(sortedLayerNames)
	sep := ""
	for _, layerName := range sortedLayerNames {
		layer := ppro.layer[layerName]
		engine.LogInfo("starting", "layer", layer.Name())
		sortedNames := layer.ParamNames()
		for _, fullName := range sortedNames {
			valstring := layer.Get(fullName)
			s += fmt.Sprintf("%s        \"%s-%s\":\"%s\"", sep, layer.Name(), fullName, valstring)
			sep = ",\n"
		}
	}
	s += sep + ppro.params.JsonValues()
	/*
		for name := range ppro.params.values {
			val := ppro.params.Get(name)
			s += fmt.Sprintf("%s        \"%s\":\"%s\"", sep, name, val)
			sep = ",\n"
		}
	*/
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}

func (ppro *PalettePro) addLayer(ctx *engine.PluginContext, name string) *engine.Layer {
	layer := engine.NewLayer(name)
	layer.AddListener(ctx)
	ppro.layer[name] = layer
	ppro.logic[name] = NewLayerLogic(ppro, layer)
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

func (ppro *PalettePro) cursorToLogic(ce engine.CursorEvent) *LayerLogic {
	// For the moment, the cursor to logic mapping is 1-to-1.
	// I.e. ce.Source of "a" maps to logic "a"
	logic, ok := ppro.logic[ce.Source]
	if !ok {
		return nil
	}
	return logic
}

func (ppro *PalettePro) cursorToLayer(ce engine.CursorEvent) *engine.Layer {
	// For the moment, the cursor to layer mapping is 1-to-1.
	// I.e. ce.Source of "a" maps to layer "a"
	layer, ok := ppro.layer[ce.Source]
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

func (ppro *PalettePro) publishOscAlive(ctx *engine.PluginContext, uptimesecs float64) {
	attractMode := ppro.attractModeIsOn
	if ppro.attractClient == nil {
		ppro.attractClient = osc.NewClient(engine.LocalAddress, AliveOutputPort)
	}
	msg := osc.NewMessage("/alive")
	msg.Append(float32(uptimesecs))
	msg.Append(attractMode)
	err := ppro.attractClient.Send(msg)
	if err != nil {
		engine.LogWarn("publishOscAlive", "err", err)
	}
}

func (ppro *PalettePro) doAttractAction(ctx *engine.PluginContext) {

	now := time.Now()
	dt := now.Sub(ppro.lastAttractGestureTime)
	if ppro.attractModeIsOn && dt > ppro.attractGestureDuration {
		layerNames := []string{"LA", "LB", "LC", "LD"}
		i := uint64(rand.Uint64()*99) % 4
		layer := layerNames[i]
		ppro.lastAttractGestureTime = now

		cid := fmt.Sprintf("%d", time.Now().UnixNano())

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		noteDuration := time.Second
		go ctx.GenerateCursorGestureesture(layer, cid, noteDuration, x0, y0, z0, x1, y1, z1)
		ppro.lastAttractGestureTime = now
	}

	dp := now.Sub(ppro.lastAttractRandTime)
	if ppro.attractPreset == "random" && dp > ppro.attractPresetDuration {
		ppro.loadPresetRand()
		ppro.lastAttractRandTime = now
	}
}

func (ppro *PalettePro) mmttInfo() *processInfo {

	// NOTE: it's inside a sub-directory of bin, so all the necessary .dll's are contained

	// The value of mmtt is either "kinect" or "oak"
	mmtt := engine.ConfigValueWithDefault("mmtt", "kinect")
	fullpath := filepath.Join(engine.PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !engine.FileExists(fullpath) {
		engine.LogWarn("no mmtt executable found, looking for", "path", fullpath)
		fullpath = ""
	}
	return NewProcessInfo("mmtt_"+mmtt+".exe", fullpath, "", nil)
}

func (ppro *PalettePro) guiInfo() *processInfo {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(engine.PaletteDir(), "bin", "pyinstalled", exe)
	return NewProcessInfo(exe, fullpath, "", nil)
}
