package engine

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2/drivers"
)

type AgentFunc func(agent *AgentContext, api string, apiargs map[string]string) (string, error)

type AgentContext struct {
	api           AgentFunc
	cursorManager *CursorManager
	params        *ParamValues
	sources       map[string]bool
}

func (agent *AgentContext) Uptime() float64 {
	return Uptime()
}

func (agent *AgentContext) GetArgsXYZ(args map[string]string) (x, y, z float32, err error) {
	return GetArgsXYZ(args)
}

func (agent *AgentContext) ClearCursors() {
	agent.cursorManager.clearCursors()
}

func (agent *AgentContext) LogInfo(msg string, keysAndValues ...any) {
	LogInfo(msg, keysAndValues...)
}

func (agent *AgentContext) LogWarn(msg string, keysAndValues ...any) {
	LogWarn(msg, keysAndValues...)
}

func (agent *AgentContext) LogError(err error, keysAndValues ...any) {
	LogError(err, keysAndValues...)
}

func (agent *AgentContext) PaletteDir() string {
	return PaletteDir()
}

func (agent *AgentContext) ConfigValueWithDefault(name, dflt string) string {
	return ConfigValueWithDefault(name, dflt)
}

func (agent *AgentContext) ConfigBoolWithDefault(name string, dflt bool) bool {
	return ConfigBoolWithDefault(name, dflt)
}

func (agent *AgentContext) ConfigValue(name string) string {
	return ConfigValue(name)
}

func (agent *AgentContext) ConfigFilePath(name string) string {
	return ConfigFilePath(name)
}

func (agent *AgentContext) StartExecutableLogOutput(logName string, fullexe string, background bool, args ...string) error {
	return StartExecutableLogOutput(logName, fullexe, background, args...)
}

func (agent *AgentContext) KillExecutable(exe string) {
	KillExecutable(exe)
}

func (agent *AgentContext) IsRunningExecutable(exe string) bool {
	return IsRunningExecutable(exe)
}

func (agent *AgentContext) FileExists(path string) bool {
	return FileExists(path)
}

func (agent *AgentContext) GetLayer(layerName string) *Layer {
	return GetLayer(layerName)
}

func (agent *AgentContext) AllowSource(source ...string) {
	var ok bool
	for _, name := range source {
		_, ok = agent.sources[name]
		if ok {
			LogInfo("AllowSource: already set?", "source", name)
		} else {
			agent.sources[name] = true
		}
	}
}

func (agent *AgentContext) IsSourceAllowed(source string) bool {
	_, ok := agent.sources[source]
	return ok
}

func (agent *AgentContext) MidiEventToPhrase(me MidiEvent) (*Phrase, error) {
	pe, err := agent.midiEventToPhraseElement(me)
	if err != nil {
		return nil, err
	}
	phr := NewPhrase().InsertElement(pe)
	return phr, nil
}

func (agent *AgentContext) midiEventToPhraseElement(me MidiEvent) (*PhraseElement, error) {

	bytes := me.Msg.Bytes()
	lng := len(bytes)
	if lng == 0 {
		err := fmt.Errorf("no bytes in midi.Message")
		LogError(err)
		return nil, err
	}
	if lng == 1 {
		return &PhraseElement{Value: NewNoteSystem(bytes[0])}, nil
	}

	var data0, data1, data2 byte
	// NOTE: channel on incoming MIDI is ignored
	// it uses whatever sound the ctx is using.
	status := bytes[0] & 0xf0
	if lng == 2 {
		switch status {
		case 0xd0: // channel pressure
			return &PhraseElement{Value: NewChanPressure(bytes[0], bytes[1])}, nil
		}

	}

	switch len(bytes) {
	case 0:
	case 1:
		data0 = bytes[0]
		status = bytes[0] & 0xf0
	case 2:
		data0 = bytes[0]
		status = bytes[0]
		data1 = bytes[1]
	case 3:
		data0 = bytes[0]
		status = bytes[0]
		data1 = bytes[1]
		data2 = bytes[2]
	default:
	}

	//	if (status == 0x90 || status == 0x80) && ctx.MIDIThruScadjust {
	//		scale := ctx.getScale()
	//		pitch = scale.ClosestTo(pitch)
	//	}

	var val any
	switch status {
	case 0x90:
		val = NewNoteOn(data0, data1, data2)
	case 0x80:
		val = NewNoteOff(data0, data1, data2)
	case 0xB0:
		val = NewController(data0, data1, data2)
	case 0xC0:
		val = NewProgramChange(data0, data1)
	case 0xD0:
		val = NewChanPressure(data0, data1)
	case 0xE0:
		val = NewPitchBend(data0, data1, data2)
	default:
		err := fmt.Errorf("MidiEventToPhraseElement: unable to handle status=0x%02x", status)
		LogError(err, "status", status)
		return nil, err
	}

	return &PhraseElement{Value: val}, nil
}

func (agent *AgentContext) CurrentClick() Clicks {
	return CurrentClick()
}

func (agent *AgentContext) SchedulePhrase(phr *Phrase, click Clicks, dest string) {
	if phr == nil {
		LogWarn("EngineContext.SchedulePhrase: phr == nil?")
		return
	}
	// ctx.SchedulePhraseAt(phr, CurrentClick())
	go func() {
		se := &SchedElement{
			AtClick: click,
			Value:   phr,
		}
		TheEngine().Scheduler.cmdInput <- SchedulerElementCmd{se}
	}()
}

func (agent *AgentContext) SubmitCommand(command Command) {
	go func() {
		TheEngine().Scheduler.cmdInput <- command
	}()
}

func (agent *AgentContext) ParamIntValue(name string) int {
	return agent.params.ParamIntValue(name)
}

func (agent *AgentContext) ParamBoolValue(name string) bool {
	return agent.params.ParamBoolValue(name)
}

/*
func (ctx *EngineContext) GetScale() *Scale {
	return ctx.scale
}

func (ctx *EngineContext) LayerParams(layerName string) *ParamValues {
	params, ok := ctx.layerParams[layerName]
	if !ok {
		params = NewParamValues()
		ctx.layerParams[layerName] = params
	}
	return params
}
*/

/*
func (ctx *EngineContext) setOneLayerParamValue(layerName, fullname, value string) error {

	layer := GetLayer(layerName)
	params := layer.params
	err := params.SetParamValueWithString(fullname, value, nil)
	if err != nil {
		return err
	}

	if strings.HasPrefix(fullname, "visual.") {
		name := strings.TrimPrefix(fullname, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, value)
		msg.Append(args)
		layer.toFreeFramePlugin(msg)
	}

	if strings.HasPrefix(fullname, "effect.") {
		name := strings.TrimPrefix(fullname, "effect.")
		// Effect parameters get sent to Resolume
		ctx.sendEffectParam(layer, name, value)
	}

	return nil
}
*/

/*
func (ctx *EngineContext) generateSpriteFromPhraseElement(pe *PhraseElement) {

	var channel uint8
	var pitch uint8
	var velocity uint8

	switch v := pe.Value.(type) {
	case *NoteOn:
		channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteOff:
		channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteFull:
		channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	default:
		return
	}

	pitchmin := uint8(ctx.agentParams.ParamIntValue("sound.pitchmin"))
	pitchmax := uint8(ctx.agentParams.ParamIntValue("sound.pitchmax"))
	if pitch < pitchmin || pitch > pitchmax {
		Warn("Unexpected value", "pitch", pitch)
		return
	}

	var x float32
	var y float32
	switch ctx.agentParams.ParamStringValue("visual.placement", "random") {
	case "random":
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

	layer := string("ABCDEFGH"[channel])
	ctx.toFreeFramePluginForLayer(layer, msg)
}
*/

// to avoid unused warning
// var p *Layer

/*
// getScale xxx
func (ctx *EngineContext) getScale() *Scale {
	var scaleName string
	var scale *Scale

	// if Layer.MIDIUseScale {
	// 	scale = Layer.externalScale
	// } else {
	scaleName = ctx.agentParams.ParamStringValue("misc.scale", "newage")
	scale = GlobalScale(scaleName)
	// }
	return scale
}
*/

/*
func (ctx *EngineContext) sendEffectParam(name string, value string) {
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		ctx.sendPadOneEffectParam(effectName, paramName, value)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			LogError(err)
			onoff = false
		}
		ctx.sendPadOneEffectOnOff(name, onoff)
	}
}
*/

func (agent *AgentContext) handleMIDITimeReset() {
	LogWarn("HandleMIDITimeReset!! needs implementation")
}

/*
func (ctx *EngineContext) sendANO() {
	if !TheRouter().generateSound {
		return
	}
	synth := ctx.agentParams.ParamStringValue("sound.synth", defaultSynth)
	SendANOToSynth(synth)
}
*/

/*
func (ctx *EngineContext) LayerApplyPreset(layerName, presetName string) error {
	preset, err := LoadPreset(presetName)
	if err != nil {
		LogError(err, "preset", presetName)
		return err
	}
	preset.ApplyTo(ctx.LayerParams(layerName))
	return fmt.Errorf("LayerApplyPreset needs work")
}
*/

func (agent *AgentContext) OpenMIDIOutput(name string) drivers.In {
	return nil
}

// GetPreset is guaranteed to return non=nil
func (agent *AgentContext) GetPreset(presetName string) *Preset {
	preset := GetPreset(presetName)
	return preset
}

/*
// LoadPreset is guaranteed to return non=nil
func (agent *Agent) LoadPreset(presetName string) *Preset {
	preset := GetPreset(presetName)
		preset, err = LoadPreset("")
		if err != nil {
			LogError(err)
		}
	}
}
*/

/*
func (ctx *EngineContext) ApplyPreset(presetName string) error {
	preset, err := LoadPreset(presetName)
	if err != nil {
		return err
	}
	return preset.ApplyTo(ctx.agentParams)
}
*/

// ExecuteAPI xxx
func (agent *AgentContext) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	DebugLogOfType("api", "Agent.ExecutAPI called", "api", api, "args", args)

	// The caller can provide rawargs if it's already known, but if not provided, we create it
	// if rawargs == "" {
	// 	rawargs = MapString(args)
	// }

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if strings.HasPrefix(api, "visual.") {
		LogInfo("ExecuteAPI: visual.* apis need work")
		/*
			msg := osc.NewMessage("/api")
			msg.Append(strings.TrimPrefix(api, "visual."))
			msg.Append(rawargs)
			layer := "A"
			ctx.toFreeFramePluginForLayer(layer, msg)
		*/
	}

	switch api {

	//
	// case "send":
	// ctx.sendAllParameters()
	// return "", err

	//	case "loop_recording":
	//		v, e := needBoolArg("onoff", api, args)
	//		if e == nil {
	//			ctx.loopIsRecording = v
	//		} else {
	//			err = e
	//		}
	//
	//	case "loop_playing":
	//		v, e := needBoolArg("onoff", api, args)
	//		if e == nil && v != layer.loopIsPlaying {
	//			layer.loopIsPlaying = v
	//			TheEngine().Scheduler.SendAllPendingNoteoffs()
	//		} else {
	//			err = e
	//		}
	//
	//	case "loop_clear":
	//		// layer.loop.Clear()
	//		// layer.clearGraphics()
	//		layer.sendANO()
	//
	//	case "loop_length":
	//		i, e := needIntArg("value", api, args)
	//		if e == nil {
	//			nclicks := Clicks(i)
	//			if nclicks != layer.loopLength {
	//				layer.loopLength = nclicks
	//				// layer.loopSetLength(nclicks)
	//			}
	//		} else {
	//			err = e
	//		}
	//
	//	case "loop_fade":
	//		f, e := needFloatArg("fade", api, args)
	//		if e == nil {
	//			layer.fadeLoop = f
	//		} else {
	//			err = e
	//		}
	//
	//	case "ANO":
	//		layer.sendANO()

	/*
		case "midi_thru":
			v, e := needBoolArg("onoff", api, args)
			if e == nil {
				layer.MIDIThru = v
			} else {
				err = e
			}

		case "midi_setscale":
			v, e := needBoolArg("onoff", api, args)
			if e == nil {
				layer.MIDISetScale = v
			} else {
				err = e
			}

		case "midi_usescale":
			v, e := needBoolArg("onoff", api, args)
			if e == nil {
				layer.MIDIUseScale = v
			} else {
				err = e
			}

		case "clearexternalscale":
			layer.clearExternalScale()
			layer.MIDINumDown = 0

		case "midi_quantized":
			v, err := needBoolArg("onoff", api, args)
			if err == nil {
				layer.MIDIQuantized = v
			}

		case "midi_thruscadjust":
			v, e := needBoolArg("onoff", api, args)
			if e == nil {
				layer.MIDIThruScadjust = v
			} else {
				err = e
			}

		case "set_transpose":
			v, e := needIntArg("value", api, args)
			if e == nil {
				layer.TransposePitch = v
				Info("layer API set_transpose", "pitch", v)
			} else {
				err = e
			}
	*/

	default:
		err = fmt.Errorf("Layer.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}

/*
func (ctx *EngineContext) sendAllParameters() {
	for nm := range ctx.agentParams.values {
		val, err := ctx.agentParams.paramValueAsString(nm)
		if err != nil {
			LogError(err)
			// Don't fail completely
			continue
		}
		// This assumes that if you set a parameter to the same value,
		// that it will re-send the mesasges to Resolume for visual.* params
		err = SetOneParamValue(ctx.agentParams, nm, val)
		if err != nil {
			LogError(err)
			// Don't fail completely
		}
	}
}
*/

func ApplyParamsMap(presetType string, paramsmap map[string]any, params *ParamValues) error {

	// Currently, no errors are ever returned, but log messages are generated.

	for name, ival := range paramsmap {
		val, okval := ival.(string)
		if !okval {
			LogWarn("value isn't a string in params json", "name", name, "value", val)
			continue
		}
		fullname := name
		thisCategory, _ := PresetNameSplit(fullname)
		// Only include ones that match the presetType
		if presetType != "snap" && thisCategory != presetType {
			continue
		}
		// This is where the parameter values get applied,
		// which may trigger things (like sending OSC)
		err := params.Set(fullname, val)
		if err != nil {
			LogError(err)
			// Don't abort the whole load, i.e. we are tolerant
			// of unknown parameters or errors in the preset
		}
	}
	return nil
}

/*
func (ctx *EngineContext) restoreCurrentSnap(layerName string) {
	preset, err := LoadPreset("snap._Current_" + layerName)
	if err != nil {
		LogError(err)
		return
	}
	err = preset.ApplyTo(ctx.agentParams)
	if err != nil {
		LogError(err)
	}
}
*/

func (agent *AgentContext) SaveCurrentAsPreset(presetName string) error {
	preset := agent.GetPreset(presetName)
	path := preset.WritableFilePath()
	return agent.SaveCurrentSnapInPath(path)
}

func (agent *AgentContext) GenerateCursorGestureesture(source string, cid string, noteDuration time.Duration, x0, y0, z0, x1, y1, z1 float32) {
	agent.cursorManager.generateCursorGestureesture(source, cid, noteDuration, x0, y0, z0, x1, y1, z1)
}

func (agent *AgentContext) SaveCurrentSnapInPath(path string) error {

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := agent.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		valstring, e := agent.params.paramValueAsString(fullName)
		if e != nil {
			LogError(e)
			continue
		}
		s += fmt.Sprintf("%s        \"%s\":\"%s\"", sep, fullName, valstring)
		sep = ",\n"
	}
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}
