package engine

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2/drivers"
)

type Task struct {
	// None of these get exported, only the methods
	methods       TaskMethods
	scheduler     *Scheduler
	cursorManager *CursorManager
	params        *ParamValues
	sources       map[string]bool
}

type TaskMethods interface {
	Start(task *Task)
	OnEvent(task *Task, e Event)
	Api(task *Task, api string, apiargs map[string]string) (string, error)
	Stop(task *Task)
}

// type TaskFunc func(ctx context.Context, e Event) (string, error)

func (task *Task) Uptime() float64 {
	return Uptime()
}

func (task *Task) LogInfo(msg string, keysAndValues ...any) {
	LogInfo(msg, keysAndValues...)
}

func (task *Task) LogWarn(msg string, keysAndValues ...any) {
	LogWarn(msg, keysAndValues...)
}

func (task *Task) LogError(err error, keysAndValues ...any) {
	LogError(err, keysAndValues...)
}

/*
func (task *Task) GetLayer(layerName string) *Layer {
	return GetLayer(layerName)
}
*/

func (task *Task) AllowSource(source ...string) {
	var ok bool
	for _, name := range source {
		_, ok = task.sources[name]
		if ok {
			LogInfo("AllowSource: already set?", "source", name)
		} else {
			task.sources[name] = true
		}
	}
}

func (task *Task) IsSourceAllowed(source string) bool {
	_, ok := task.sources[source]
	return ok
}

func (task *Task) MidiEventToPhrase(me MidiEvent) (*Phrase, error) {
	pe, err := task.midiEventToPhraseElement(me)
	if err != nil {
		return nil, err
	}
	phr := NewPhrase().InsertElement(pe)
	return phr, nil
}

func (task *Task) midiEventToPhraseElement(me MidiEvent) (*PhraseElement, error) {

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

func (task *Task) CurrentClick() Clicks {
	return CurrentClick()
}

func (task *Task) ScheduleDebug() string {
	return fmt.Sprintf("%s", task.scheduler)
}

func (task *Task) SchedulePhrase(phr *Phrase, click Clicks, dest string) {
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
		task.scheduler.cmdInput <- SchedulerElementCmd{se}
	}()
}

func (task *Task) SubmitCommand(command Command) {
	go func() {
		task.scheduler.cmdInput <- Command{"attractmode", true}
	}()
}

func (task *Task) ParamIntValue(name string) int {
	return task.params.ParamIntValue(name)
}

func (task *Task) ParamBoolValue(name string) bool {
	return task.params.ParamBoolValue(name)
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

// getEffectMap returns the resolume.json map for a given effect
// and map type ("on", "off", or "params")
func getEffectMap(effectName string, mapType string) (map[string]any, string, int, error) {
	if effectName[1] != '-' {
		err := fmt.Errorf("no dash in effect, name=%s", effectName)
		return nil, "", 0, err
	}
	effects, ok := ResolumeJSON["effects"]
	if !ok {
		err := fmt.Errorf("no effects value in resolume.json?")
		return nil, "", 0, err
	}
	realEffectName := effectName[2:]

	n, err := strconv.Atoi(effectName[0:1])
	if err != nil {
		return nil, "", 0, fmt.Errorf("bad format of effectName=%s", effectName)
	}
	effnum := int(n)

	effectsmap := effects.(map[string]any)
	oneEffect, ok := effectsmap[realEffectName]
	if !ok {
		err := fmt.Errorf("no effects value for effect=%s", effectName)
		return nil, "", 0, err
	}
	oneEffectMap := oneEffect.(map[string]any)
	mapValue, ok := oneEffectMap[mapType]
	if !ok {
		err := fmt.Errorf("no params value for effect=%s", effectName)
		return nil, "", 0, err
	}
	return mapValue.(map[string]any), realEffectName, effnum, nil
}

func addLayerAndClipNums(addr string, layerNum int, clipNum int) string {
	if addr[0] != '/' {
		LogWarn("addr in resolume.json doesn't start with /", "addr", addr)
		addr = "/" + addr
	}
	addr = fmt.Sprintf("/composition/layers/%d/clips/%d/video/effects%s", layerNum, clipNum, addr)
	return addr
}

func resolumeEffectNameOf(name string, num int) string {
	if num == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, num)
}

/*
func (ctx *EngineContext) sendPadOneEffectParam(effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
	paramsMap, realEffectName, realEffectNum, err := layer.getEffectMap(effectName, "params")
	if err != nil {
		LogError(err)
		return
	}
	if paramsMap == nil {
		Warn("No params value for", "effecdt", effectName)
		return
	}
	oneParam, ok := paramsMap[paramName]
	if !ok {
		Warn("No params value for", "param", paramName, "effect", effectName)
		return
	}

	oneDef, ok := ParamDefs[fullName]
	if !ok {
		Warn("No paramdef value for", "param", paramName, "effect", effectName)
		return
	}

	addr := oneParam.(string)
	resEffectName := ctx.resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)
	addr = addLayerAndClipNums(addr, layer.resolumeLayer, 1)

	msg := osc.NewMessage(addr)

	// Append the value to the message, depending on the type of the parameter

	switch oneDef.typedParamDef.(type) {

	case paramDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			LogError(err)
			valint = 0
		}
		msg.Append(int32(valint))

	case paramDefBool:
		valbool, err := strconv.ParseBool(value)
		if err != nil {
			LogError(err)
			valbool = false
		}
		onoffValue := 0
		if valbool {
			onoffValue = 1
		}
		msg.Append(int32(onoffValue))

	case paramDefString:
		valstr := value
		msg.Append(valstr)

	case paramDefFloat:
		var valfloat float32
		valfloat, err := ParseFloat32(value, resEffectName)
		if err != nil {
			LogError(err)
			valfloat = 0.0
		}
		msg.Append(float32(valfloat))

	default:
		Warn("SetParamValueWithString: unknown type of ParamDef for", "name", fullName)
		return
	}

	ctx.toResolume(msg)
}
*/

func (task *Task) handleMIDITimeReset() {
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

func (task *Task) OpenMIDIOutput(name string) drivers.In {
	return nil
}

// GetPreset is guaranteed to return non=nil
func (task *Task) GetPreset(presetName string) *Preset {
	preset := GetPreset(presetName)
	return preset
}

/*
// LoadPreset is guaranteed to return non=nil
func (task *Task) LoadPreset(presetName string) *Preset {
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
func (task *Task) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	DebugLogOfType("api", "Agent.ExecutAPI called", "api", api, "args", args)
	// The caller can provide rawargs if it's already known, but if not provided, we create it
	if rawargs == "" {
		rawargs = MapString(args)
	}

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

func (task *Task) SaveCurrentAsPreset(presetName string) error {
	preset := task.GetPreset(presetName)
	path := preset.WritableFilePath()
	return task.SaveCurrentSnapInPath(path)
}

func (task *Task) GenerateCursorGestureesture(source string, cid string, noteDuration time.Duration, x0, y0, z0, x1, y1, z1 float32) {
	task.cursorManager.generateCursorGestureesture(source, cid, noteDuration, x0, y0, z0, x1, y1, z1)
}

func (task *Task) SaveCurrentSnapInPath(path string) error {

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := task.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		valstring, e := task.params.paramValueAsString(fullName)
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
