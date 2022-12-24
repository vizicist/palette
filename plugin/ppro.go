package plugin

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

var AliveOutputPort = 3331

type PalettePro struct {
	started        bool
	layer          map[string]*engine.Layer
	logic          map[string]*LayerLogic
	resolume       *Resolume
	bidule         *Bidule
	processManager *ProcessManager

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
	lastAttractPresetTime  time.Time
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

type LayerLogic struct {
	ppro         *PalettePro
	layer        *engine.Layer
	synth        *engine.Synth
	lastActiveID int

	// tempoFactor            float64
	// loop            *StepLoop
	loopLength      engine.Clicks
	loopIsRecording bool
	loopIsPlaying   bool
	fadeLoop        float32
	// lastCursorStepEvent    CursorStepEvent
	// lastUnQuantizedStepNum Clicks

	// params                         map[string]any

	// paramsMutex               sync.RWMutex
	activeNotes      map[string]*ActiveNote
	activeNotesMutex sync.RWMutex

	// Things moved over from Router

	// MIDINumDown      int
	// MIDIThru         bool
	// MIDIThruScadjust bool
	// MIDISetScale     bool
	// MIDIQuantized    bool
	// MIDIUseScale     bool // if true, scadjust uses "external" Scale
	// TransposePitch   int
	// externalScale    *Scale
}

func init() {
	transposebeats := engine.Clicks(engine.ConfigIntWithDefault("transposebeats", 48))
	ppro := &PalettePro{
		layer:           map[string]*engine.Layer{},
		logic:           map[string]*LayerLogic{},
		attractModeIsOn: false,
		generateVisuals: engine.ConfigBoolWithDefault("generatevisuals", true),
		generateSound:   engine.ConfigBoolWithDefault("generatesound", true),

		lastAttractCommand:     time.Time{},
		lastAttractGestureTime: time.Time{},
		lastAttractPresetTime:  time.Time{},
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
		return "", fmt.Errorf("ppro.stop doesn't do anything")

	case "onparamset":
		layerName, ok := apiargs["layer"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing layer argument")
		}
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing name argument")
		}
		value, ok := apiargs["value"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing value argument")
		}
		ppro.onParamSet(ctx, layerName, name, value)
		return "", nil

	case "echo":
		value, ok := apiargs["value"]
		if !ok {
			value = "ECHO!"
		}
		return value, nil

	case "startprocess":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("ExecuteAPI: missing process argument")
		} else {
			err = ppro.processManager.StartRunning(process)
		}
		return "", err

	case "stopprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("ExecuteAPI: missing process argument")
		} else {
			return "", ppro.processManager.StopRunning(process)
		}

	case "activate":
		for _, pi := range ppro.processManager.info {
			if pi.Activate != nil {
				pi.Activate()
			}
		}
		return "", nil

	case "killall":
		ppro.processManager.killAll()
		return "", nil

	case "event":
		eventName, ok := apiargs["event"]
		if !ok {
			return "", fmt.Errorf("PalettePro: Missing event argument")
		}
		switch eventName {
		case "click":
			return ppro.onClick(ctx, apiargs)
		case "midi":
			return ppro.onMidiEvent(ctx, apiargs)
		case "cursor":
			return ppro.onCursorEvent(ctx, apiargs)
		default:
			return "", fmt.Errorf("PalettePro: Unhandled event type %s", eventName)
		}

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

	default:
		ctx.LogWarn("Pro.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}

	// return result, err
}

func (ppro *PalettePro) start(ctx *engine.PluginContext) error {

	if ppro.started {
		return fmt.Errorf("PalettePro: already started")
	}
	ppro.started = true
	ppro.resolume = NewResolume(ctx)
	ppro.bidule = NewBidule(ctx)

	ppro.processManager = NewProcessManager(ctx)

	ppro.processManager.AddProcess("resolume", ppro.resolume.ProcessInfo())
	ppro.processManager.AddProcess("bidule", ppro.bidule.ProcessInfo())
	ppro.processManager.AddProcess("gui", ppro.guiInfo())
	ppro.processManager.AddProcess("mmtt", ppro.mmttInfo())

	ctx.AllowSource("A", "B", "C", "D")

	layerA := ppro.addLayer(ctx, "a")
	layerA.Set("visual.shape", "circle")
	layerA.Apply(ctx.GetPreset("snap.White_Ghosts"))

	layerB := ppro.addLayer(ctx, "b")
	layerB.Set("visual.shape", "square")
	layerB.Apply(ctx.GetPreset("snap.Concentric_Squares"))

	layerC := ppro.addLayer(ctx, "c")
	layerC.Set("visual.shape", "square")
	layerC.Apply(ctx.GetPreset("snap.Circular_Moire"))

	layerD := ppro.addLayer(ctx, "d")
	layerD.Set("visual.shape", "square")
	layerD.Apply(ctx.GetPreset("snap.Diagonal_Mirror"))

	//ctx.ApplyPreset("quad.Quick Scat_Circles")

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

func (ppro *PalettePro) onParamSet(ctx *engine.PluginContext, layerName string, paramName string, paramValue string) {

	layer, ok := ppro.layer[layerName]
	if !ok {
		ctx.LogWarn("No layer named", "layer", layerName)
		return
	}

	if strings.HasPrefix(paramName, "visual.") {
		name := strings.TrimPrefix(paramName, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, paramValue)
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
			ctx.LogWarn("PalettePro: no synth named", "synth", paramValue)
		}
		ppro.logic[layerName].synth = synth
	}

}

func (ppro *PalettePro) onSpriteGen(layerName string, id string, x, y, z float32) {
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	ppro.resolume.toFreeFramePlugin(layerName, msg)
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
			ctx.LogWarn("PalettePro.OnClick: should be turning attractmode on")
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
			ppro.checkProcessesAndRestartIfNecessary()
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

	/*
		// XXX - this should be done in the erae plugin
			if EraeEnabled {
				HandleEraeMIDI(me)
			}
	*/

	if ppro.MIDIThru {
		ctx.LogWarn("PassThruMIDI needs work")
		// ppro.PassThruMIDI(e)
	}
	if ppro.MIDISetScale {
		ppro.handleMIDISetScaleNote(me)
	}

	ctx.LogInfo("PalettePro.onMidiEvent", "me", me)
	phr, err := ctx.MidiEventToPhrase(me)
	if err != nil {
		return "", err
	}
	if phr != nil {
		ctx.SchedulePhrase(phr, ctx.CurrentClick(), "P_04_C_04")
	}
	return "", nil
}

func (ppro *PalettePro) saveCurrentParams(ctx *engine.PluginContext) {
	for _, layer := range ppro.layer {
		err := layer.SaveCurrentParams()
		if err != nil {
			ctx.LogError(err)
		}
	}
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
			ctx.LogInfo("PalettePro: shouold be turning attract mode OFF")
			ppro.lastAttractCommand = time.Now()
		}

	}

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

	return "", nil
}
func (ppro *PalettePro) loadQuadPresetRand(ctx *engine.PluginContext) {

	arr, err := engine.PresetArray("quad")
	if err != nil {
		ctx.LogError(err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	ctx.LogInfo("loadQuadPresetRand", "preset", arr[rn])
	preset := ctx.GetPreset(arr[rn])
	ppro.loadQuadPreset(ctx, preset)
	if err != nil {
		ctx.LogError(err)
	}
}

func (ppro *PalettePro) loadQuadPreset(ctx *engine.PluginContext, preset *engine.Preset) {
	for _, layer := range ppro.layer {
		layer.ApplyQuadPreset(preset)
	}
}

func (ppro *PalettePro) addLayer(ctx *engine.PluginContext, name string) *engine.Layer {
	layer := engine.NewLayer(name)
	layer.AddListener(ctx)
	ppro.layer[name] = layer
	ppro.logic[name] = &LayerLogic{
		layer: layer,
		ppro:  ppro,
	}
	return layer
}

func (ppro *PalettePro) scheduleNoteNow(ctx *engine.PluginContext, dest string, pitch, velocity uint8, duration engine.Clicks) {
	ctx.LogInfo("PalettePro.scheculeNoteNow", "dest", dest, "pitch", pitch)
	pe := &engine.PhraseElement{Value: engine.NewNoteFull(0, pitch, velocity, duration)}
	phr := engine.NewPhrase().InsertElement(pe)
	phr.Destination = dest
	ctx.SchedulePhrase(phr, ctx.CurrentClick(), dest)
}

/*
func (ppro *PalettePro) channelToDestination(channel int) string {
	return fmt.Sprintf("P_03_C_%02d", channel)
}
*/

func (ppro *PalettePro) cursorToLayer(ce engine.CursorEvent) *engine.Layer {
	// For the moment, the Source to layer mapping is 1-to-1.
	// and the corresponding layers are abcd.
	switch ce.Source {
	case "A":
		return ppro.layer["a"]
	case "B":
		return ppro.layer["b"]
	case "C":
		return ppro.layer["c"]
	case "D":
		return ppro.layer["d"]
	default:
		return nil
	}
}

func (logic *LayerLogic) cursorToNoteOn(ctx *engine.PluginContext, ce engine.CursorEvent) *engine.NoteOn {
	pitch := logic.cursorToPitch(ctx, ce)
	velocity := logic.cursorToVelocity(ctx, ce)
	channel := logic.cursorToChannel(ctx, ce)
	return engine.NewNoteOn(channel, pitch, velocity)
}

func (logic *LayerLogic) cursorToChannel(ctx *engine.PluginContext, ce engine.CursorEvent) (channel uint8) {
	synth := logic.synth
	if synth == nil {
		ctx.LogWarn("cursorToChannel: No synth?")
		channel = 1
	} else {
		channel = synth.Channel()
	}
	return channel
}

func (logic *LayerLogic) cursorToPitch(ctx *engine.PluginContext, ce engine.CursorEvent) uint8 {
	layer := logic.layer
	ppro := logic.ppro
	pitchmin := layer.GetInt("sound.pitchmin")
	pitchmax := layer.GetInt("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)

	chromatic := ctx.ParamBoolValue("sound.chromatic")
	if !chromatic {
		scale := ppro.scale
		p = scale.ClosestTo(p)
		// MIDIOctaveShift might be negative
		i := int(p) + 12*ppro.MIDIOctaveShift
		for i < 0 {
			i += 12
		}
		for i > 127 {
			i -= 12
		}
		p = uint8(i + ppro.TransposePitch)
	}
	return p
}

func (logic *LayerLogic) cursorToVelocity(ctx *engine.PluginContext, ce engine.CursorEvent) uint8 {
	layer := logic.layer
	vol := layer.Get("misc.vol")
	velocitymin := layer.GetInt("sound.velocitymin")
	velocitymax := layer.GetInt("sound.velocitymax")
	// bogus, when values in json are missing
	if velocitymin == 0 && velocitymax == 0 {
		velocitymin = 0
		velocitymax = 127
	}
	if velocitymin > velocitymax {
		t := velocitymin
		velocitymin = velocitymax
		velocitymax = t
	}
	v := float32(0.8) // default and fixed value
	switch vol {
	case "frets":
		v = 1.0 - ce.Y
	case "pressure":
		v = ce.Z * 4.0
	case "fixed":
		// do nothing
	default:
		ctx.LogWarn("Unrecognized vol value", "vol", vol)
	}
	dv := velocitymax - velocitymin + 1
	p1 := int(v * float32(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}

func (ppro *PalettePro) clearExternalScale() {
	ppro.externalScale = engine.MakeScale()
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
		ctx.LogWarn("publishOscAlive", "err", err)
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

	dp := now.Sub(ppro.lastAttractPresetTime)
	if ppro.attractPreset == "random" && dp > ppro.attractPresetDuration {
		ppro.loadQuadPresetRand(ctx)
		ppro.lastAttractPresetTime = now
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
	return &processInfo{"mmtt_" + mmtt + ".exe", fullpath, "", nil}
}

func (ppro *PalettePro) guiInfo() *processInfo {
	exe := "palette_gui.exe"
	fullpath := filepath.Join(engine.PaletteDir(), "bin", "pyinstalled", exe)
	return &processInfo{exe, fullpath, "", nil}
}

func (ppro *PalettePro) checkProcessesAndRestartIfNecessary() {
	autostart := engine.ConfigValueWithDefault("autostart", "")
	if autostart == "" || autostart == "nothing" || autostart == "none" {
		return
	}
	processes := strings.Split(autostart, ",")
	pm := ppro.processManager
	for _, processName := range processes {
		p, _ := pm.getProcessInfo(processName)
		if p != nil {
			if !pm.isRunning(processName) {
				go func(name string) {
					pm.StartRunning(name)
					pm.Activate(name)
				}(processName)
			}
		}
	}

}

func (logic *LayerLogic) generateSoundFromCursor(ctx *engine.PluginContext, ce engine.CursorEvent) {
	layer := logic.layer
	a := logic.getActiveNote(ce.ID)
	switch ce.Ddu {
	case "down":
		// Send noteoff for current note
		if a.noteOn != nil {
			// I think this happens if we get things coming in
			// faster than the checkDelay can generate the UP event.
			logic.sendNoteOff(a)
		}
		a.noteOn = logic.cursorToNoteOn(ctx, ce)
		a.ce = ce
		logic.sendNoteOn(ctx, a)
	case "drag":
		if a.noteOn == nil {
			// if we turn on playing in the middle of an existing loop,
			// we may see some drag events without a down.
			// Also, I'm seeing this pretty commonly in other situations,
			// not really sure what the underlying reason is,
			// but it seems to be harmless at the moment.
			ctx.LogWarn("=============== HEY! drag event, a.currentNoteOn == nil?\n")
			return
		}
		newNoteOn := logic.cursorToNoteOn(ctx, ce)
		oldpitch := a.noteOn.Pitch
		newpitch := newNoteOn.Pitch
		// We only turn off the existing note (for a given Cursor ID)
		// and start the new one if the pitch changes

		// Also do this if the Z/Velocity value changes more than the trigger value

		// NOTE: this could and perhaps should use a.ce.Z now that we're
		// saving a.ce, like the deltay value

		dz := float64(int(a.noteOn.Velocity) - int(newNoteOn.Velocity))
		deltaz := float32(math.Abs(dz) / 128.0)
		deltaztrig := layer.GetFloat("sound._deltaztrig")

		deltay := float32(math.Abs(float64(a.ce.Y - ce.Y)))
		deltaytrig := layer.GetFloat("sound._deltaytrig")

		if layer.Get("sound.controllerstyle") == "modulationonly" {
			zmin := layer.GetFloat("sound._controllerzmin")
			zmax := layer.GetFloat("sound._controllerzmax")
			cmin := layer.GetInt("sound._controllermin")
			cmax := layer.GetInt("sound._controllermax")
			oldz := a.ce.Z
			newz := ce.Z
			// XXX - should put the old controller value in ActiveNote so
			// it doesn't need to be computed every time
			oldzc := BoundAndScaleController(oldz, zmin, zmax, cmin, cmax)
			newzc := BoundAndScaleController(newz, zmin, zmax, cmin, cmax)

			if newzc != 0 && newzc != oldzc {
				logic.synth.SendController(1, newzc)
			}
		}

		if newpitch != oldpitch || deltaz > deltaztrig || deltay > deltaytrig {
			logic.sendNoteOff(a)
			a.noteOn = newNoteOn
			a.ce = ce
			logic.sendNoteOn(ctx, a)
		}
	case "up":
		if a.noteOn == nil {
			// not sure why this happens, yet
			ctx.LogWarn("Unexpected UP when currentNoteOn is nil?", "layer", layer.Name())
		} else {
			logic.sendNoteOff(a)

			a.noteOn = nil
			a.ce = ce // Hmmmm, might be useful, or wrong
		}
		logic.activeNotesMutex.Lock()
		delete(logic.activeNotes, ce.ID)
		logic.activeNotesMutex.Unlock()
	}
}

// ActiveNote is a currently active MIDI note
type ActiveNote struct {
	id     int
	noteOn *engine.NoteOn
	ce     engine.CursorEvent // the one that triggered the note
}

func (logic *LayerLogic) getActiveNote(id string) *ActiveNote {
	logic.activeNotesMutex.RLock()
	a, ok := logic.activeNotes[id]
	logic.activeNotesMutex.RUnlock()
	if !ok {
		logic.lastActiveID++
		a = &ActiveNote{
			id:     logic.lastActiveID,
			noteOn: nil,
		}
		logic.activeNotesMutex.Lock()
		logic.activeNotes[id] = a
		logic.activeNotesMutex.Unlock()
	}
	return a
}

func (logic *LayerLogic) terminateActiveNotes(ctx *engine.PluginContext) {
	logic.activeNotesMutex.RLock()
	for id, a := range logic.activeNotes {
		if a != nil {
			logic.sendNoteOff(a)
		} else {
			ctx.LogWarn("Hey, nil activeNotes entry for", "id", id)
		}
	}
	logic.activeNotesMutex.RUnlock()
}

func (logic *LayerLogic) clearGraphics() {
	// send an OSC message to Resolume
	logic.ppro.resolume.toFreeFramePlugin(logic.layer.Name(), osc.NewMessage("/clear"))
}

func (logic *LayerLogic) nextQuant(t engine.Clicks, q engine.Clicks) engine.Clicks {
	// the algorithm below is the same as KeyKit's nextquant
	if q <= 1 {
		return t
	}
	tq := t
	rem := tq % q
	if (rem * 2) > q {
		tq += (q - rem)
	} else {
		tq -= rem
	}
	if tq < t {
		tq += q
	}
	return tq
}

func (logic *LayerLogic) sendNoteOn(ctx *engine.PluginContext, a *ActiveNote) {

	pe := &engine.PhraseElement{Value: a.noteOn}
	logic.SendPhraseElementToSynth(ctx, pe)

	ss := logic.layer.Get("visual.spritesource")
	if ss == "midi" {
		logic.generateSpriteFromPhraseElement(ctx, pe)
	}
}

func (logic *LayerLogic) generateSprite(id string, x, y, z float32) {

	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	logic.ppro.resolume.toFreeFramePlugin(logic.layer.Name(), msg)
}

func (logic *LayerLogic) SendPhraseElementToSynth(ctx *engine.PluginContext, pe *engine.PhraseElement) {

	ss := logic.layer.Get("visual.spritesource")
	if ss == "midi" {
		logic.generateSpriteFromPhraseElement(ctx, pe)
	}
	logic.synth.SendTo(pe)
}

func (logic *LayerLogic) generateSpriteFromPhraseElement(ctx *engine.PluginContext, pe *engine.PhraseElement) {

	layer := logic.layer

	// var channel uint8
	var pitch uint8
	var velocity uint8

	switch v := pe.Value.(type) {
	case *engine.NoteOn:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *engine.NoteOff:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *engine.NoteFull:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	default:
		return
	}

	pitchmin := uint8(layer.GetInt("sound.pitchmin"))
	pitchmax := uint8(layer.GetInt("sound.pitchmax"))
	if pitch < pitchmin || pitch > pitchmax {
		ctx.LogWarn("Unexpected value", "pitch", pitch)
		return
	}

	var x float32
	var y float32
	switch layer.Get("visual.placement") {
	case "random", "":
		x = rand.Float32()
		y = rand.Float32()
	case "linear":
		y = 0.5
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "cursor":
		x = rand.Float32()
		y = rand.Float32()
	case "top":
		y = 1.0
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "bottom":
		y = 0.0
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "left":
		y = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 0.0
	case "right":
		y = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 1.0
	default:
		x = rand.Float32()
		y = rand.Float32()
	}

	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(float32(velocity) / 127.0)

	// Someday localhost should be changed to the actual IP address.
	// XXX - Set sprite ID to pitch, is this right?
	msg.Append(fmt.Sprintf("%d@localhost", pitch))

	logic.ppro.resolume.toFreeFramePlugin(layer.Name(), msg)
}

func (logic *LayerLogic) sendNoteOff(a *ActiveNote) {
	n := a.noteOn
	if n == nil {
		// Not sure why this sometimes happens
		return
	}
	noteOff := engine.NewNoteOff(n.Channel, n.Pitch, n.Velocity)
	pe := &engine.PhraseElement{Value: noteOff}
	logic.synth.SendTo(pe)
	// layer.SendPhraseElementToSynth(pe)
}

func (logic *LayerLogic) advanceTransposeTo(newclick engine.Clicks) {

	ppro := logic.ppro
	ppro.transposeNext += (ppro.transposeClicks * engine.OneBeat)
	ppro.transposeIndex = (ppro.transposeIndex + 1) % len(ppro.transposeValues)
	/*
			transposePitch := ppro.transposeValues[ppro.transposeIndex]
				fr _, layer := range TheRouter().layers {
					// layer.clearDown()
					LogOfType("transpose""setting transposepitch in layer","pad", layer.padName, "transposePitch",transposePitch, "nactive",len(layer.activeNotes))
					layer.TransposePitch = transposePitch
				}
		sched.SendAllPendingNoteoffs()
	*/
}

func BoundAndScaleController(v, vmin, vmax float32, cmin, cmax int) int {
	newv := BoundAndScaleFloat(v, vmin, vmax, float32(cmin), float32(cmax))
	return int(newv)
}

func BoundAndScaleFloat(v, vmin, vmax, outmin, outmax float32) float32 {
	if v < vmin {
		v = vmin
	} else if v > vmax {
		v = vmax
	}
	out := outmin + (outmax-outmin)*((v-vmin)/(vmax-vmin))
	return out
}
