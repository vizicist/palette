package agent

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

var AliveOutputPort = 3331

type PalettePro struct {
	started        bool
	layer          map[string]*engine.Layer
	resolume       *Resolume
	bidule         *Bidule
	processManager *ProcessManager

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
}

func init() {
	ppro := &PalettePro{
		layer:                  map[string]*engine.Layer{},
		attractModeIsOn:        false,
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
	}
	ppro.clearExternalScale()
	RegisterAgent("ppro", ppro.Api)
}
func (ppro *PalettePro) Api(ctx *engine.AgentContext, api string, apiargs map[string]string) (result string, err error) {

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
		ppro.onParamSet(layerName, name, value)
		return "", nil

	case "onspritegen":
		layerName, ok := apiargs["layer"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing layer argument")
		}
		id, ok := apiargs["id"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing id argument")
		}
		x, ok := apiargs["x"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing x argument")
		}
		y, ok := apiargs["y"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing y argument")
		}
		z, ok := apiargs["z"]
		if !ok {
			return "", fmt.Errorf("PalettePro.Api: Missing z argument")
		}
		xf, err := strconv.ParseFloat(x, 32)
		if err != nil {
			return "", fmt.Errorf("PalettePro.Api: Bad format for x argument")
		}
		yf, err := strconv.ParseFloat(y, 32)
		if err != nil {
			return "", fmt.Errorf("PalettePro.Api: Bad format for y argument")
		}
		zf, err := strconv.ParseFloat(z, 64)
		if err != nil {
			return "", fmt.Errorf("PalettePro.Api: Bad format for z argument")
		}
		ppro.onSpriteGen(layerName, id, float32(xf), float32(yf), float32(zf))
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

func (ppro *PalettePro) start(ctx *engine.AgentContext) error {

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

func (ppro *PalettePro) onParamSet(layerName string, paramName string, paramValue string) {

	layer := engine.GetLayer(layerName)
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
}

func (ppro *PalettePro) onSpriteGen(layerName string, id string, x, y, z float32) {
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	ppro.resolume.toFreeFramePlugin(layerName, msg)
}

func (ppro *PalettePro) onClick(ctx *engine.AgentContext, apiargs map[string]string) (string, error) {

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

func (ppro *PalettePro) onMidiEvent(ctx *engine.AgentContext, apiargs map[string]string) (string, error) {

	me, err := engine.MidiEventFromMap(apiargs)
	if err != nil {
		return "", err
	}

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

func (ppro *PalettePro) saveCurrentSnaps(ctx *engine.AgentContext) {
	for _, layer := range ppro.layer {
		err := layer.SaveCurrentSnap()
		if err != nil {
			ctx.LogError(err)
		}
	}
}

func (ppro *PalettePro) onCursorEvent(ctx *engine.AgentContext, apiargs map[string]string) (string, error) {

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
	msg := osc.NewMessage("/sprite")
	msg.Append(ce.X)
	msg.Append(ce.Y)
	msg.Append(ce.Z)
	msg.Append(ce.ID)
	ppro.resolume.toFreeFramePlugin(layer.Name(), msg)

	return "", nil
}

func (ppro *PalettePro) loadQuadPresetRand(ctx *engine.AgentContext) {

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

func (ppro *PalettePro) loadQuadPreset(ctx *engine.AgentContext, preset *engine.Preset) {
	for layerName, layer := range ppro.layer {
		layer.ApplyQuadPreset(preset, layerName)
	}
}

func (ppro *PalettePro) addLayer(ctx *engine.AgentContext, name string) *engine.Layer {
	layer := engine.NewLayer(name)
	layer.AddListener(ctx)
	ppro.layer[name] = layer
	return layer
}

func (ppro *PalettePro) scheduleNoteNow(ctx *engine.AgentContext, dest string, pitch, velocity uint8, duration engine.Clicks) {
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
	return ppro.layer["a"]
}

func (ppro *PalettePro) cursorToPitch(ctx *engine.AgentContext, ce engine.CursorEvent) uint8 {
	layer := ppro.cursorToLayer(ce)
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

func (ppro *PalettePro) publishOscAlive(ctx *engine.AgentContext, uptimesecs float64) {
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

func (ppro *PalettePro) doAttractAction(ctx *engine.AgentContext) {

	now := time.Now()
	dt := now.Sub(ppro.lastAttractGestureTime)
	if ppro.attractModeIsOn && dt > ppro.attractGestureDuration {
		layerNames := []string{"A", "B", "C", "D"}
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