package agent

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

var AliveOutputPort = 3331

type PalettePro struct {
	layer map[string]*engine.Layer

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
	scale                  *engine.Scale
}

func init() {
	ppro := &PalettePro{
		attractModeIsOn:        false,
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
		scale:                  nil,
	}
	RegisterTask("palettepro", ppro)
}

func (ppro *PalettePro) Start(task *engine.Task) {

	engine.Info("PalettePro.Start")

	task.AllowSource("A", "B", "C", "D")

	a := task.GetLayer("a")
	a.Set("visual.ffglport", "3334")
	a.Set("visual.shape", "circle")
	a.Apply(task.GetPreset("snap.White_Ghosts"))

	b := task.GetLayer("b")
	b.Set("visual.ffglport", "3335")
	b.Set("visual.shape", "square")
	b.Apply(task.GetPreset("snap.Concentric_Squares"))

	c := task.GetLayer("c")
	c.Set("visual.ffglport", "3336")
	c.Set("visual.shape", "square")
	c.Apply(task.GetPreset("snap.Circular_Moire"))

	d := task.GetLayer("d")
	d.Set("visual.resolumeport", "3337")
	d.Set("visual.shape", "square")
	d.Apply(task.GetPreset("snap.Diagonal_Mirror"))

	//ctx.ApplyPreset("quad.Quick Scat_Circles")

	ppro.attractCheckSecs = float64(engine.ConfigFloatWithDefault("attractchecksecs", 2))
	ppro.attractIdleSecs = float64(engine.ConfigFloatWithDefault("attractidlesecs", 0))

	secs1 := engine.ConfigFloatWithDefault("attractpresetduration", 30)
	ppro.attractPresetDuration = time.Duration(int(secs1 * float32(time.Second)))

	secs := engine.ConfigFloatWithDefault("attractgestureduration", 0.5)
	ppro.attractGestureDuration = time.Duration(int(secs * float32(time.Second)))

	secs = engine.ConfigFloatWithDefault("attractnoteduration", 0.2)
	ppro.attractNoteDuration = time.Duration(int(secs * float32(time.Second)))

	ppro.attractPreset = engine.ConfigStringWithDefault("attractpreset", "random")

}

func (ppro *PalettePro) Stop(task *engine.Task) {
}

func (ppro *PalettePro) OnEvent(task *engine.Task, event engine.Event) {
	task.LogInfo("PalettePro.OnEvent", "event", event)
	switch e := event.(type) {
	case engine.ClickEvent:
		ppro.OnClick(task, e)
	case engine.MidiEvent:
		ppro.OnMidiEvent(task, e)
	case engine.CursorEvent:
		ppro.OnCursorEvent(task, e)
	}
}

func (ppro *PalettePro) OnClick(task *engine.Task, ce engine.ClickEvent) {
	uptimesecs := task.Uptime()
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
			task.LogWarn("PalettePro.OnClick: attract needs work")
			// go func() {
			// 	task.cmdInput <- Command{"attractmode", true}
			// }()
			// sched.SetAttractMode(true)
		}
	}

	if ppro.attractModeIsOn {
		ppro.doAttractAction(task)
	}

	sinceLastAlive := uptimesecs - ppro.lastAlive
	if sinceLastAlive > ppro.aliveSecs {
		ppro.publishOscAlive(task, uptimesecs)
		ppro.lastAlive = uptimesecs
	}
}

func (ppro *PalettePro) OnMidiEvent(task *engine.Task, me engine.MidiEvent) {

	//if r.ctx.MIDIThru {
	//layer.PassThruMIDI(e)
	//}
	//if layer.MIDISetScale {
	//r.handleMIDISetScaleNote(e)
	//}

	task.LogInfo("PalettePro.onMidiEvent", "me", me)
	phr, err := task.MidiEventToPhrase(me)
	if err != nil {
		engine.LogError(err)
	}
	if phr != nil {
		task.SchedulePhrase(phr, task.CurrentClick(), "P_04_C_04")
	}
}

func (ppro *PalettePro) Api(task *engine.Task, api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "nextalive":
		// acts like a timer, but it could wait for
		// some event if necessary
		time.Sleep(2 * time.Second)
		js := engine.JsonObject(
			"event", "alive",
			"seconds", fmt.Sprintf("%f", task.Uptime()),
			"attractmode", fmt.Sprintf("%v", ppro.attractModeIsOn),
		)
		return js, nil

	case "load":
		presetName, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		preset, err := engine.LoadPreset(presetName)
		if err != nil {
			return "", err
		}

		layerName, okLayer := apiargs["layer"]
		if !okLayer {
			layerName = "*"
		}

		if preset.Category == "quad" {
			// The layerName might be only a single layer, and loadQuadPreset
			// will only load that one layer from the quad preset
			if layerName == "*" {
				return "", fmt.Errorf("load of quad needs work")
			}
			layer := ppro.layer[layerName]
			err = layer.ApplyQuadPreset(preset, layerName)
			if err != nil {
				task.LogError(err)
				return "", err
			}
			ppro.SaveCurrentSnaps(task)
		} else {
			// It's a non-quad preset for a single layer.
			// However, the layerName can still be "*" to apply to all layers.
			layer := ppro.layer[layerName]
			layer.Apply(preset)
			ppro.SaveCurrentSnaps(task)
		}
		return "", err

	case "save":
		// return "", fmt.Errorf("executePresetAPI needs work")
		presetName, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		return "", task.SaveCurrentAsPreset(presetName)

	default:
		task.LogWarn("Router.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}
}

func (ppro *PalettePro) SaveCurrentSnaps(task *engine.Task) {
	for _, layer := range ppro.layer {
		err := layer.SaveCurrentSnap()
		if err != nil {
			task.LogError(err)
		}
	}
}

func (ppro *PalettePro) loadQuadPresetRand(task *engine.Task) {

	arr, err := engine.PresetArray("quad")
	if err != nil {
		task.LogError(err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	task.LogInfo("loadQuadPresetRand", "preset", arr[rn])
	preset := task.GetPreset(arr[rn])
	ppro.loadQuadPreset(task, preset)
	if err != nil {
		task.LogError(err)
	}
}

func (ppro *PalettePro) loadQuadPreset(task *engine.Task, preset *engine.Preset) {
	for layerName, layer := range ppro.layer {
		layer.ApplyQuadPreset(preset, layerName)
	}
}

func (ppro *PalettePro) OnCursorEvent(task *engine.Task, ce engine.CursorEvent) {

	if ce.Ddu == "down" { // || ce.Ddu == "drag" {
		engine.Info("OnCursorEvent", "ce", ce)
		layer := ppro.cursorToLayer(ce)
		pitch := ppro.cursorToPitch(task, ce)
		velocity := uint8(ce.Z * 1280)
		duration := 4 * engine.QuarterNote
		dest := layer.Get("sound.synth")
		ppro.scheduleNoteNow(task, dest, pitch, velocity, duration)
	}
}

func (ppro *PalettePro) scheduleNoteNow(task *engine.Task, dest string, pitch, velocity uint8, duration engine.Clicks) {
	engine.Info("PalettePro.scheculeNoteNow", "dest", dest, "pitch", pitch)
	pe := &engine.PhraseElement{Value: engine.NewNoteFull(0, pitch, velocity, duration)}
	phr := engine.NewPhrase().InsertElement(pe)
	phr.Destination = dest
	task.SchedulePhrase(phr, task.CurrentClick(), dest)
}

func (ppro *PalettePro) channelToDestination(channel int) string {
	return fmt.Sprintf("P_03_C_%02d", channel)
}

func (ppro *PalettePro) cursorToLayer(ce engine.CursorEvent) *engine.Layer {
	return ppro.layer["a"]
}

func (ppro *PalettePro) cursorToPitch(task *engine.Task, ce engine.CursorEvent) uint8 {
	a := ppro.layer["a"]
	pitchmin := a.GetInt("sound.pitchmin")
	pitchmax := a.GetInt("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)

	// layer := task.GetLayer("a")

	chromatic := task.ParamBoolValue("sound.chromatic")
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

func (ppro *PalettePro) publishOscAlive(task *engine.Task, uptimesecs float64) {
	attractMode := ppro.attractModeIsOn
	if ppro.attractClient == nil {
		ppro.attractClient = osc.NewClient(engine.LocalAddress, AliveOutputPort)
	}
	msg := osc.NewMessage("/alive")
	msg.Append(float32(uptimesecs))
	msg.Append(attractMode)
	err := ppro.attractClient.Send(msg)
	if err != nil {
		task.LogWarn("publishOscAlive", "err", err)
	}
}

func (ppro *PalettePro) doAttractAction(task *engine.Task) {

	now := time.Now()
	dt := now.Sub(ppro.lastAttractGestureTime)
	if ppro.attractModeIsOn && dt > ppro.attractGestureDuration {
		playerNames := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		player := playerNames[i]
		ppro.lastAttractGestureTime = now

		cid := fmt.Sprintf("%d", time.Now().UnixNano())

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		noteDuration := time.Second
		go task.GenerateCursorGestureesture(player, cid, noteDuration, x0, y0, z0, x1, y1, z1)
		ppro.lastAttractGestureTime = now
	}

	dp := now.Sub(ppro.lastAttractPresetTime)
	if ppro.attractPreset == "random" && dp > ppro.attractPresetDuration {
		ppro.loadQuadPresetRand(task)
		ppro.lastAttractPresetTime = now
	}
}
