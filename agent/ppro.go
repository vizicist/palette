package agent

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

var AliveOutputPort = 3331

type PalettePro struct {
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
	RegisterAgent("ppro", ppro)
}

func (ppro *PalettePro) Start(agent *engine.Agent) error {

	ppro.resolume = NewResolume(agent)
	ppro.bidule = NewBidule(agent)

	ppro.processManager = NewProcessManager(agent)

	ppro.processManager.AddProcess("resolume", ppro.resolume.ProcessInfo())
	ppro.processManager.AddProcess("bidule", ppro.bidule.ProcessInfo())
	ppro.processManager.AddProcess("gui", ppro.guiInfo())
	ppro.processManager.AddProcess("mmtt", ppro.mmttInfo())

	agent.AllowSource("A", "B", "C", "D")

	layerA := ppro.addLayer(agent, "a")
	layerA.Set("visual.shape", "circle")
	layerA.Apply(agent.GetPreset("snap.White_Ghosts"))

	layerB := ppro.addLayer(agent, "b")
	layerB.Set("visual.shape", "square")
	layerB.Apply(agent.GetPreset("snap.Concentric_Squares"))

	layerC := ppro.addLayer(agent, "c")
	layerC.Set("visual.shape", "square")
	layerC.Apply(agent.GetPreset("snap.Circular_Moire"))

	layerD := ppro.addLayer(agent, "d")
	layerD.Set("visual.shape", "square")
	layerD.Apply(agent.GetPreset("snap.Diagonal_Mirror"))

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

func (ppro *PalettePro) OnParamSet(layer *engine.Layer, paramName string, paramValue string) {

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

func (ppro *PalettePro) OnSpriteGen(layer *engine.Layer, id string, x, y, z float32) {
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	ppro.resolume.toFreeFramePlugin(layer.Name(), msg)
}

func (ppro *PalettePro) addLayer(agent *engine.Agent, name string) *engine.Layer {
	layer := engine.NewLayer(name, ppro)
	ppro.layer[name] = layer
	return layer
}

func (ppro *PalettePro) Stop(agent *engine.Agent) {
}

func (ppro *PalettePro) OnEvent(agent *engine.Agent, event engine.Event) {
	switch e := event.(type) {
	case engine.ClickEvent:
		ppro.OnClick(agent, e)
	case engine.MidiEvent:
		ppro.OnMidiEvent(agent, e)
	case engine.CursorEvent:
		ppro.OnCursorEvent(agent, e)
	default:
		engine.LogWarn("PalettePro: Unhandled event type", "event", event)
	}
}

func (ppro *PalettePro) OnClick(agent *engine.Agent, ce engine.ClickEvent) {
	uptimesecs := agent.Uptime()
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
			agent.LogWarn("PalettePro.OnClick: should be turning attractmode on")
			ppro.lastAttractCommand = time.Now()
		}
	}

	if ppro.attractModeIsOn {
		ppro.doAttractAction(agent)
	}

	sinceLastAlive := uptimesecs - ppro.lastAlive
	if sinceLastAlive > ppro.aliveSecs {
		ppro.publishOscAlive(agent, uptimesecs)
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

}

func (ppro *PalettePro) OnMidiEvent(agent *engine.Agent, me engine.MidiEvent) {

	if ppro.MIDIThru {
		agent.LogWarn("PassThruMIDI needs work")
		// ppro.PassThruMIDI(e)
	}
	if ppro.MIDISetScale {
		ppro.handleMIDISetScaleNote(me)
	}

	agent.LogInfo("PalettePro.onMidiEvent", "me", me)
	phr, err := agent.MidiEventToPhrase(me)
	if err != nil {
		agent.LogError(err)
	}
	if phr != nil {
		agent.SchedulePhrase(phr, agent.CurrentClick(), "P_04_C_04")
	}
}

func (ppro *PalettePro) Api(agent *engine.Agent, api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "echo":
		value, ok := apiargs["value"]
		if !ok {
			value = "ECHO!"
		}
		result = value

	case "start":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("ExecuteAPI: missing process argument")
		} else {
			err = ppro.processManager.StartRunning(process)
		}

	case "stop":
		process, ok := apiargs["process"]
		if !ok {
			err = fmt.Errorf("ExecuteAPI: missing process argument")
		} else {
			err = ppro.processManager.StopRunning(process)
		}

	case "activate":
		for _, pi := range ppro.processManager.info {
			if pi.Activate != nil {
				pi.Activate()
			}
		}

	case "killall":
		ppro.processManager.killAll()

	case "event":
		event, ok := apiargs["event"]
		if !ok {
			err = fmt.Errorf("PalettePro.Api: Missing value argument")
			break
		}

		switch event {
		case "cursor_down", "cursor_drag", "cursor_up":
			x, y, z, e := engine.GetArgsXYZ(apiargs)
			if e != nil {
				err = e
			}
			agent.LogInfo("PalettePro.API: xyz=%f,%f,%f", x, y, z)

		default:
			agent.LogWarn("PalettePro.API: unhandled api=%s", api)
			err = fmt.Errorf("unhandled event=%s", event)
		}

	case "nextalive":
		// acts like a timer, but it could wait for
		// some event if necessary
		time.Sleep(2 * time.Second)
		result = engine.JsonObject(
			"event", "alive",
			"seconds", fmt.Sprintf("%f", agent.Uptime()),
			"attractmode", fmt.Sprintf("%v", ppro.attractModeIsOn),
		)

	default:
		agent.LogWarn("Pro.ExecuteAPI api is not recognized\n", "api", api)
		err = fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}

	return result, err
}

func (ppro *PalettePro) SaveCurrentSnaps(agent *engine.Agent) {
	for _, layer := range ppro.layer {
		err := layer.SaveCurrentSnap()
		if err != nil {
			agent.LogError(err)
		}
	}
}

func (ppro *PalettePro) OnCursorEvent(agent *engine.Agent, ce engine.CursorEvent) {

	if ce.Ddu == "down" { // || ce.Ddu == "drag" {
		agent.LogInfo("OnCursorEvent", "ce", ce)
		layer := ppro.cursorToLayer(ce)
		pitch := ppro.cursorToPitch(agent, ce)
		velocity := uint8(ce.Z * 1280)
		duration := 3 * engine.QuarterNote
		dest := layer.Get("sound.synth")
		ppro.scheduleNoteNow(agent, dest, pitch, velocity, duration)
	}

	// Any non-internal cursor will turn attract mode off.
	if ce.Source != "internal" {
		if time.Since(ppro.lastAttractCommand) > time.Second {
			agent.LogInfo("PalettePro: shouold be turning attract mode OFF")
			ppro.lastAttractCommand = time.Now()
		}

	}
}

func (ppro *PalettePro) loadQuadPresetRand(agent *engine.Agent) {

	arr, err := engine.PresetArray("quad")
	if err != nil {
		agent.LogError(err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	agent.LogInfo("loadQuadPresetRand", "preset", arr[rn])
	preset := agent.GetPreset(arr[rn])
	ppro.loadQuadPreset(agent, preset)
	if err != nil {
		agent.LogError(err)
	}
}

func (ppro *PalettePro) loadQuadPreset(agent *engine.Agent, preset *engine.Preset) {
	for layerName, layer := range ppro.layer {
		layer.ApplyQuadPreset(preset, layerName)
	}
}

func (ppro *PalettePro) scheduleNoteNow(agent *engine.Agent, dest string, pitch, velocity uint8, duration engine.Clicks) {
	agent.LogInfo("PalettePro.scheculeNoteNow", "dest", dest, "pitch", pitch)
	pe := &engine.PhraseElement{Value: engine.NewNoteFull(0, pitch, velocity, duration)}
	phr := engine.NewPhrase().InsertElement(pe)
	phr.Destination = dest
	agent.SchedulePhrase(phr, agent.CurrentClick(), dest)
}

/*
func (ppro *PalettePro) channelToDestination(channel int) string {
	return fmt.Sprintf("P_03_C_%02d", channel)
}
*/

func (ppro *PalettePro) cursorToLayer(ce engine.CursorEvent) *engine.Layer {
	return ppro.layer["a"]
}

func (ppro *PalettePro) cursorToPitch(agent *engine.Agent, ce engine.CursorEvent) uint8 {
	layer := ppro.cursorToLayer(ce)
	pitchmin := layer.GetInt("sound.pitchmin")
	pitchmax := layer.GetInt("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)

	// layer := agent.GetLayer("a")

	chromatic := agent.ParamBoolValue("sound.chromatic")
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

func (ppro *PalettePro) publishOscAlive(agent *engine.Agent, uptimesecs float64) {
	attractMode := ppro.attractModeIsOn
	if ppro.attractClient == nil {
		ppro.attractClient = osc.NewClient(engine.LocalAddress, AliveOutputPort)
	}
	msg := osc.NewMessage("/alive")
	msg.Append(float32(uptimesecs))
	msg.Append(attractMode)
	err := ppro.attractClient.Send(msg)
	if err != nil {
		agent.LogWarn("publishOscAlive", "err", err)
	}
}

func (ppro *PalettePro) doAttractAction(agent *engine.Agent) {

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
		go agent.GenerateCursorGestureesture(layer, cid, noteDuration, x0, y0, z0, x1, y1, z1)
		ppro.lastAttractGestureTime = now
	}

	dp := now.Sub(ppro.lastAttractPresetTime)
	if ppro.attractPreset == "random" && dp > ppro.attractPresetDuration {
		ppro.loadQuadPresetRand(agent)
		ppro.lastAttractPresetTime = now
	}
}
