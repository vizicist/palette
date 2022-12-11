package agent

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

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
		attractIdleSecs:        0,
	})
	RegisterTask("palettepro", PalettePro.OnEvent, ppro)
}

type PalettePro struct {
	ctx   *engine.TaskContext
	layer map[string]*engine.Layer

	MIDIOctaveShift  int
	MIDINumDown      int
	MIDIThru         bool
	MIDIThruScadjust bool
	MIDISetScale     bool
	MIDIQuantized    bool
	MIDIUseScale     bool // if true, scadjust uses "external" Scale
	TransposePitch   int
	// externalScale    *engine.Scale

	attractModeIsOn        bool
	lastAttractGestureTime time.Time
	lastAttractPresetTime  time.Time
	attractGestureDuration time.Duration
	attractNoteDuration    time.Duration
	attractPresetDuration  time.Duration
	attractPreset          string
	attractClient          *osc.Client
	lastAttractChange      float64
	attractCheckSecs       float64
	attractIdleSecs        float64
}

func (agent *PalettePro) Start(ctx *engine.TaskContext) {

	engine.Info("PalettePro.Start")

	ctx.AllowSource("A", "B", "C", "D")

	a := ctx.GetLayer("a")
	a.Set("visual.ffglport", "3334")
	a.Set("visual.shape", "circle")
	a.Apply(ctx.GetPreset("snap.White_Ghosts"))

	b := ctx.GetLayer("b")
	b.Set("visual.ffglport", "3335")
	b.Set("visual.shape", "square")
	b.Apply(ctx.GetPreset("snap.Concentric_Squares"))

	c := ctx.GetLayer("c")
	c.Set("visual.ffglport", "3336")
	c.Set("visual.shape", "square")
	c.Apply(ctx.GetPreset("snap.Circular_Moire"))

	d := ctx.GetLayer("d")
	d.Set("visual.resolumeport", "3337")
	d.Set("visual.shape", "square")
	d.Apply(ctx.GetPreset("snap.Diagonal_Mirror"))

	ctx.ApplyPreset("quad.Quick Scat_Circles")

	agent.attractCheckSecs = float64(ConfigFloatWithDefault("attractchecksecs", 2))
	agent.attractIdleSecs = float64(ConfigFloatWithDefault("attractidlesecs", 0))

	secs1 := ConfigFloatWithDefault("attractpresetduration", 30)
	agent.attractPresetDuration = time.Duration(int(secs1 * float32(time.Second)))

	secs := ConfigFloatWithDefault("attractgestureduration", 0.5)
	agent.attractGestureDuration = time.Duration(int(secs * float32(time.Second)))

	secs = ConfigFloatWithDefault("attractnoteduration", 0.2)
	agent.attractNoteDuration = time.Duration(int(secs * float32(time.Second)))

	agent.attractPreset = ConfigStringWithDefault("attractpreset", "random")

	var lastAttractCheck float64

	agent.ctx = ctx
}

func (ppro *PalettePro) OnEvent(me engine.Event) {
	engine.Info("Agent_processes.OnEvent", "me", me)
	switch e := event.(type) {
	case engine.ClickEvent:
		agent.OnClick(e)
	case engine.MidiEvent:
		agent.OnMidiEvent(e)
	case engine.CursorEvent:
		agent.OnCursorEvent(e)
	}
}
func (agent *PalettePro) OnClick(ce engine.ClickEvent) {
	// Every so often we check to see if attract mode should be turned on
	attractModeEnabled := agent.attractIdleSecs > 0
	sinceLastAttractChange := uptimesecs - agent.lastAttractChange
	sinceLastAttractCheck := uptimesecs - lastAttractCheck
	if attractModeEnabled && sinceLastAttractCheck > agent.attractCheckSecs {
		lastAttractCheck = uptimesecs
		// There's a delay when checking cursor activity to turn attract mod on.
		// Non-internal cursor activity turns attract mode off instantly.
		if !agent.attractModeIsOn && sinceLastAttractChange > agent.attractIdleSecs {
			// Nothing happening for a while, turn attract mode on
			go func() {
				agent.cmdInput <- Command{"attractmode", true}
			}()
			// sched.SetAttractMode(true)
		}
	}

	if agent.attractModeIsOn {
		agent.doAttractAction()
	}

}

func (agent *PalettePro) OnMidiEvent(me engine.MidiEvent) {

	if agent.ctx == nil {
		engine.LogError(fmt.Errorf("OnMidiEvent: Start needs to be called before this"))
		return
	}

	ctx := agent.ctx

	/*
		if r.ctx.MIDIThru {
			layer.PassThruMIDI(e)
		}
		if layer.MIDISetScale {
			r.handleMIDISetScaleNote(e)
		}
	*/
	ctx.Log("PalettePro.onMidiEvent", "me", me)
	phr, err := ctx.MidiEventToPhrase(me)
	if err != nil {
		engine.LogError(err)
	}
	if phr != nil {
		ctx.SchedulePhrase(phr, ctx.CurrentClick(), "P_04_C_04")
	}
}

func (agent *PalettePro) Api(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "nextalive":
		// acts like a timer, but it could wait for
		// some event if necessary
		time.Sleep(2 * time.Second)
		js := JsonObject(
			"event", "alive",
			"seconds", fmt.Sprintf("%f", e.Scheduler.aliveSecs),
			"attractmode", fmt.Sprintf("%v", e.Scheduler.attractModeIsOn),
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
			layer := agent.layer[layerName]
			err = layer.applyQuadPreset(preset)
			if err != nil {
				engine.LogError(err)
				return "", err
			}
			agent.ctx.SaveCurrentSnaps(layerName)
		} else {
			// It's a non-quad preset for a single layer.
			// However, the layerName can still be "*" to apply to all layers.
			err = preset.ApplyTo(layerName)
			if err != nil {
				LogError(err, "presetName", presetName)
			} else {
				agent.ctx.SaveCurrentSnaps(layerName)
			}
		}
		return "", err

	case "save":
		return "", fmt.Errorf("executePresetAPI needs work")
		/*
			presetName, okpreset := apiargs["preset"]
			if !okpreset {
				return "", fmt.Errorf("missing preset parameter")
			}
			taskName, okplayer := apiargs["agent"]
			if !okplayer {
				return "", fmt.Errorf("missing agent parameter")
			}
			ctx, err := e.Router.taskManager.GetEngineContext(taskName)
			if err != nil {
				return "", err
			}
			return "", ctx.saveCurrentAsPreset(presetName)
		*/

	default:
		Warn("Router.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}

	switch api {

	case "load":
		return "", fmt.Errorf("executePresetAPI needs work")
		/*
			presetName, okpreset := apiargs["preset"]
			if !okpreset {
				return "", fmt.Errorf("missing preset parameter")
			}
			preset, err := LoadPreset(presetNeame)
			if err != nil {
				return "", err
			}

			taskName, okLayer := apiargs["agent"]
			if !okLayer {
				taskName = "*"
			}
			if preset.category == "quad" {
				// The layerName might be only a single layer, and loadQuadPreset
				// will only load that one layer from the quad preset
				err = preset.applyQuadPresetToLayer(layerName)
				if err != nil {
					LogError(err)
					return "", err
				}
				e.Router.SaveCurrentSnaps(layerName)
			} else {
				// It's a non-quad preset for a single layer.
				// However, the layerName can still be "*" to apply to all layers.
				err = preset.ApplyTo(layerName)
				if err != nil {
					LogError(err, "presetName", presetName)
				} else {
					e.Router.SaveCurrentSnaps(layerName)
				}
			}
			return "", err
		*/

	case "save":
		return "", fmt.Errorf("executePresetAPI needs work")
		/*
			presetName, okpreset := apiargs["preset"]
			if !okpreset {
				return "", fmt.Errorf("missing preset parameter")
			}
			taskName, okLayer := apiargs["agent"]
			if !okLayer {
				return "", fmt.Errorf("missing agent parameter")
			}
			ctx, err := e.Router.taskManager.GetEngineContext(taskName)
			if err != nil {
				return "", err
			}
			return "", ctx.saveCurrentAsPreset(presetName)
		*/

	default:
		engine.Warn("Router.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecutePresetAPI unrecognized api=%s", api)
	}

}

func (agent *PalettePro) loadQuadPresetRand() {

	arr, err := PresetArray("quad")
	if err != nil {
		LogError(err)
		return
	}
	rn := rand.Uint64() % uint64(len(arr))
	Info("loadQuadPresetRand", "preset", arr[rn])
	preset, err := LoadPreset(arr[rn])
	if err != nil {
		LogError(err)
	} else {
		preset.applyQuadPresetToPlayer("*")
	}
}

func (agent *PalettePro) OnCursorEvent(ce engine.CursorEvent) {

	if agent.ctx == nil {
		engine.LogError(fmt.Errorf("OnMidiEvent: Start needs to be called before this"))
		return
	}

	// ctx := agent.ctx

	if ce.Ddu == "down" { // || ce.Ddu == "drag" {
		engine.Info("OnCursorEvent", "ce", ce)
		layer := agent.cursorToLayer(ce)
		pitch := agent.cursorToPitch(ce)
		velocity := uint8(ce.Z * 1280)
		duration := 4 * engine.QuarterNote
		dest := layer.Get("sound.synth")
		agent.scheduleNoteNow(dest, pitch, velocity, duration)
	}
}

func (agent *PalettePro) scheduleNoteNow(dest string, pitch, velocity uint8, duration engine.Clicks) {
	engine.Info("PalettePro.scheculeNoteNow", "dest", dest, "pitch", pitch)
	ctx := agent.ctx
	pe := &engine.PhraseElement{Value: engine.NewNoteFull(0, pitch, velocity, duration)}
	phr := engine.NewPhrase().InsertElement(pe)
	phr.Destination = dest
	ctx.SchedulePhrase(phr, ctx.CurrentClick(), dest)
}

func (agent *PalettePro) channelToDestination(channel int) string {
	return fmt.Sprintf("P_03_C_%02d", channel)
}

func (agent *PalettePro) cursorToLayer(ce engine.CursorEvent) *engine.Layer {
	return agent.layer["a"]
}

func (agent *PalettePro) cursorToPitch(ce engine.CursorEvent) uint8 {
	a := agent.layer["a"]
	pitchmin := a.GetInt("sound.pitchmin")
	pitchmax := a.GetInt("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	/*
		chromatic := r.ctx.ParamBoolValue("sound.chromatic")
		if !chromatic {
			scale := r.ctx.GetScale()
			p = scale.ClosestTo(p)
			// MIDIOctaveShift might be negative
			i := int(p) + 12*layer.MIDIOctaveShift
			for i < 0 {
				i += 12
			}
			for i > 127 {
				i -= 12
			}
			p = uint8(i + layer.TransposePitch)
		}
	*/
	return p
}

/*
func (r *PalettePro) handleMIDISetScaleNote(e engine.MidiEvent) {
	status := e.Status() & 0xf0
	pitch := int(e.Data1())
	if status == 0x90 {
		/		// If there are no notes held down (i.e. this is the first), clear the scale
		if layer.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a noteon
			layer.MIDINumDown = 0
		}
		if layer.MIDINumDown == 0 {
			layer.clearExternalScale()
		}
		layer.setExternalScale(pitch%12, true)
		layer.MIDINumDown++
		if pitch < 60 {
			layer.MIDIOctaveShift = -1
		} else if pitch > 72 {
			layer.MIDIOctaveShift = 1
		} else {
			layer.MIDIOctaveShift = 0
		}
	} else if status == 0x80 {
		layer.MIDINumDown--
	}
}
*/

func (agent *PalettePro) publishOscAlive(uptimesecs float64) {
	attractMode := agent.attractModeIsOn
	DebugLogOfType("attract", "publishOscAlive", "uptimesecs", uptimesecs, "attract", attractMode)
	if agent.attractClient == nil {
		agent.attractClient = osc.NewClient(LocalAddress, AliveOutputPort)
	}
	msg := osc.NewMessage("/alive")
	msg.Append(float32(uptimesecs))
	msg.Append(attractMode)
	err := agent.attractClient.Send(msg)
	if err != nil {
		Warn("publishOscAlive", "err", err)
	}
}

func (agent *PalettePro) doAttractAction() {

	now := time.Now()
	dt := now.Sub(sched.lastAttractGestureTime)
	if sched.attractModeIsOn && dt > sched.attractGestureDuration {
		playerNames := []string{"A", "B", "C", "D"}
		i := uint64(rand.Uint64()*99) % 4
		player := playerNames[i]
		sched.lastAttractGestureTime = now

		cid := fmt.Sprintf("%d", time.Now().UnixNano())

		x0 := rand.Float32()
		y0 := rand.Float32()
		z0 := rand.Float32() / 2.0

		x1 := rand.Float32()
		y1 := rand.Float32()
		z1 := rand.Float32() / 2.0

		noteDuration := time.Second
		go TheRouter().cursorManager.generateCursorGestureesture(player, cid, noteDuration, x0, y0, z0, x1, y1, z1)
		sched.lastAttractGestureTime = now
	}

	dp := now.Sub(sched.lastAttractPresetTime)
	if sched.attractPreset == "random" && dp > TheEngine().Scheduler.attractPresetDuration {
		TheRouter().loadQuadPresetRand()
		sched.lastAttractPresetTime = now
	}
}
