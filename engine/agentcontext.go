package engine

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hypebeast/go-osc/osc"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type AgentContext struct {
	agent           Agent
	scheduler       *Scheduler
	agentParams     *ParamValues
	layerParams     map[string]*ParamValues
	scale           *Scale
	sources         map[string]bool
	freeframeClient *osc.Client
	resolumeClient  *osc.Client
	resolumeLayer   int // see ResolumeLayerForPad

	// state     interface{} // for future agent-specific state
}

func NewAgentContext(agent Agent) *AgentContext {
	return &AgentContext{
		scheduler:   TheEngine().Scheduler,
		agent:       agent,
		agentParams: NewParamValues(),
		layerParams: map[string]*ParamValues{},
		scale:       &Scale{},
		sources:     map[string]bool{},
	}
}

func (ctx *AgentContext) Log(msg string, keysAndValues ...interface{}) {
	Info(msg, keysAndValues...)
}

func (ctx *AgentContext) MakeLayer(name string) *Layer {
	return MakeLayer(name)
}

func (ctx *AgentContext) saveCurrentAsPrefix(presetName string) error {
	return fmt.Errorf("saveCurrentAsPrefix needs work")
}

func (ctx *AgentContext) MidiEventToPhrase(me MidiEvent) (*Phrase, error) {
	pe, err := ctx.midiEventToPhraseElement(me)
	if err != nil {
		return nil, err
	}
	phr := NewPhrase().InsertElement(pe)
	return phr, nil
}

func (ctx *AgentContext) midiEventToPhraseElement(me MidiEvent) (*PhraseElement, error) {

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

	var val interface{}
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

func (ctx *AgentContext) CurrentClick() Clicks {
	return CurrentClick()
}

func (ctx *AgentContext) ScheduleDebug() string {
	return fmt.Sprintf("%s", ctx.scheduler)
}

func (ctx *AgentContext) ScheduleNoteNow(channel, pitch, velocity uint8, duration Clicks) {
	DebugLogOfType("schedule", "ctx.ScheduleNoteNow", "pitch", pitch)
	pe := &PhraseElement{Value: NewNoteFull(channel, pitch, velocity, duration)}
	phr := NewPhrase().InsertElement(pe)
	ctx.SchedulePhraseAt(phr, CurrentClick())
}

func (ctx *AgentContext) SchedulePhraseNow(phr *Phrase) {
	ctx.SchedulePhraseAt(phr, CurrentClick())
}

func (ctx *AgentContext) SchedulePhraseAt(phr *Phrase, click Clicks) {
	if phr == nil {
		Warn("AgentContext.SchedulePhraseAt: phr == nil?")
		return
	}
	DebugLogOfType("schedule", "ctx.SchedulePhraseAt", "click", click)
	go func() {
		se := &SchedElement{
			AtClick: click,
			Value:   phr,
		}
		ctx.scheduler.cmdInput <- ScheduleElementCmd{se}
	}()
}

func (ctx *AgentContext) ParamIntValue(name string) int {
	return ctx.agentParams.ParamIntValue(name)
}

func (ctx *AgentContext) ParamBoolValue(name string) bool {
	return ctx.agentParams.ParamBoolValue(name)
}

func (ctx *AgentContext) GetScale() *Scale {
	return ctx.scale
}

func (ctx *AgentContext) LayerParams(layerName string) *ParamValues {
	params, ok := ctx.layerParams[layerName]
	if !ok {
		params = NewParamValues()
		ctx.layerParams[layerName] = params
	}
	return params
}

func SetOneParamValue(params *ParamValues, fullname, value string) error {
	return params.SetParamValueWithString(fullname, value, nil)
}

func (ctx *AgentContext) setOneLayerParamValue(layer, fullname, value string) error {

	params := ctx.LayerParams(layer)
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
		ctx.toFreeFramePluginForLayer(layer, msg)
	}

	if strings.HasPrefix(fullname, "effect.") {
		name := strings.TrimPrefix(fullname, "effect.")
		// Effect parameters get sent to Resolume
		ctx.sendEffectParam(layer, name, value)
	}

	return nil
}

func (ctx *AgentContext) sendEffectParam(layer string, name string, value string) {
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		ctx.sendPadOneEffectParam(layer, effectName, paramName, value)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			LogError(err)
			onoff = false
		}
		ctx.sendPadOneEffectOnOff(layer, name, onoff)
	}
}

func (ctx *AgentContext) sendPadOneEffectParam(layer string, effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
	paramsMap, realEffectName, realEffectNum, err := getEffectMap(effectName, "params")
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
	resEffectName := resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)
	addr = addLayerAndClipNums(addr, ctx.resolumeLayer, 1)

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

func ffglPortForLayer(layer string) int {
	layers := "ABCDEFGH"
	i := strings.Index(layers, layer)
	return 3334 + i
}

func (ctx *AgentContext) toFreeFramePluginForLayer(layer string, msg *osc.Message) {
	if ctx.freeframeClient == nil {
		ctx.freeframeClient = osc.NewClient(LocalAddress, ffglPortForLayer(layer))
	}
	ctx.freeframeClient.Send(msg)
}

func (ctx *AgentContext) toResolume(msg *osc.Message) {
	if ctx.resolumeClient == nil {
		ctx.resolumeClient = osc.NewClient(LocalAddress, ResolumePort)
	}
	ctx.resolumeClient.Send(msg)
}

func (ctx *AgentContext) generateSpriteForLayer(layer string, id string, x, y, z float32) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	ctx.toFreeFramePluginForLayer(layer, msg)
}

func (ctx *AgentContext) generateSpriteFromPhraseElement(pe *PhraseElement) {

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

// to avoid unused warning
// var p *Player

// getScale xxx
func (ctx *AgentContext) getScale() *Scale {
	var scaleName string
	var scale *Scale

	// if player.MIDIUseScale {
	// 	scale = player.externalScale
	// } else {
	scaleName = ctx.agentParams.ParamStringValue("misc.scale", "newage")
	scale = GlobalScale(scaleName)
	// }
	return scale
}

func (ctx *AgentContext) sendPadOneEffectOnOff(layer string, effectName string, onoff bool) {
	var mapType string
	if onoff {
		mapType = "on"
	} else {
		mapType = "off"
	}

	onoffMap, realEffectName, realEffectNum, err := getEffectMap(effectName, mapType)
	if err != nil {
		LogError(err)
		return
	}

	if onoffMap == nil {
		Warn("No onoffMap value for", "effect", effectName, "maptype", mapType, effectName)
		return
	}

	onoffAddr, ok := onoffMap["addr"]
	if !ok {
		Warn("No addr value in onoff", "effect", effectName)
		return
	}
	onoffArg, ok := onoffMap["arg"]
	if !ok {
		Warn("No arg valuei in onoff for", "effect", effectName)
		return
	}
	addr := onoffAddr.(string)
	addr = ctx.addEffectNum(addr, realEffectName, realEffectNum)
	addr = addLayerAndClipNums(addr, ctx.resolumeLayer, 1)
	onoffValue := int(onoffArg.(float64))

	msg := osc.NewMessage(addr)
	msg.Append(int32(onoffValue))
	ctx.toResolume(msg)
}

func (ctx *AgentContext) addEffectNum(addr string, effect string, num int) string {
	if num == 1 {
		return addr
	}
	// e.g. "blur" becomes "blur2"
	return strings.Replace(addr, effect, fmt.Sprintf("%s%d", effect, num), 1)
}

/*
func (ctx *AgentContext) sendEffectParam(name string, value string) {
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
func getEffectMap(effectName string, mapType string) (map[string]interface{}, string, int, error) {
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

	effectsmap := effects.(map[string]interface{})
	oneEffect, ok := effectsmap[realEffectName]
	if !ok {
		err := fmt.Errorf("no effects value for effect=%s", effectName)
		return nil, "", 0, err
	}
	oneEffectMap := oneEffect.(map[string]interface{})
	mapValue, ok := oneEffectMap[mapType]
	if !ok {
		err := fmt.Errorf("no params value for effect=%s", effectName)
		return nil, "", 0, err
	}
	return mapValue.(map[string]interface{}), realEffectName, effnum, nil
}

func addLayerAndClipNums(addr string, layerNum int, clipNum int) string {
	if addr[0] != '/' {
		Warn("addr in resolume.json doesn't start with /", "addr", addr)
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
func (ctx *AgentContext) sendPadOneEffectParam(effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
	paramsMap, realEffectName, realEffectNum, err := player.getEffectMap(effectName, "params")
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
	addr = addLayerAndClipNums(addr, player.resolumeLayer, 1)

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

func (ctx *AgentContext) HandleMIDITimeReset() {
	Warn("HandleMIDITimeReset!! needs implementation")
}

func (ctx *AgentContext) sendANO() {
	if !TheRouter().generateSound {
		return
	}
	synth := ctx.agentParams.ParamStringValue("sound.synth", defaultSynth)
	SendANOToSynth(synth)
}

func (ctx *AgentContext) AllowSource(source ...string) {
	var ok bool
	for _, name := range source {
		_, ok = ctx.sources[name]
		if ok {
			Info("AllowSource: already set?", "source", name)
		} else {
			ctx.sources[name] = true
		}
	}
}

/*
func (ctx *AgentContext) LayerApplyPreset(layerName, presetName string) error {
	preset, err := LoadPreset(presetName)
	if err != nil {
		LogError(err, "preset", presetName)
		return err
	}
	preset.ApplyTo(ctx.LayerParams(layerName))
	return fmt.Errorf("LayerApplyPreset needs work")
}
*/

func (ctx *AgentContext) OpenMIDIOutput(name string) drivers.In {
	return nil
}

// GetPreset is guaranteed to return non=nil
func (ctx *AgentContext) GetPreset(presetName string) *Preset {
	preset, err := LoadPreset(presetName)
	if err != nil {
		LogError(err)
		preset, err = LoadPreset("")
		if err != nil {
			LogError(err)
		}
	}
	return preset
}

func (ctx *AgentContext) ApplyPreset(presetName string) error {
	preset, err := LoadPreset(presetName)
	if err != nil {
		return err
	}
	return preset.ApplyTo(ctx.agentParams)
}

// ExecuteAPI xxx
func (ctx *AgentContext) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	DebugLogOfType("api", "Agent.ExecutAPI called", "api", api, "args", args)
	// The caller can provide rawargs if it's already known, but if not provided, we create it
	if rawargs == "" {
		rawargs = MapString(args)
	}

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if strings.HasPrefix(api, "visual.") {
		msg := osc.NewMessage("/api")
		msg.Append(strings.TrimPrefix(api, "visual."))
		msg.Append(rawargs)
		layer := "A"
		ctx.toFreeFramePluginForLayer(layer, msg)
	}

	switch api {

	case "send":
		ctx.sendAllParameters()
		return "", err

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
		//		if e == nil && v != player.loopIsPlaying {
		//			player.loopIsPlaying = v
		//			TheEngine().Scheduler.SendAllPendingNoteoffs()
		//		} else {
		//			err = e
		//		}
		//
		//	case "loop_clear":
		//		// player.loop.Clear()
		//		// player.clearGraphics()
		//		player.sendANO()
		//
		//	case "loop_length":
		//		i, e := needIntArg("value", api, args)
		//		if e == nil {
		//			nclicks := Clicks(i)
		//			if nclicks != player.loopLength {
		//				player.loopLength = nclicks
		//				// player.loopSetLength(nclicks)
		//			}
		//		} else {
		//			err = e
		//		}
		//
		//	case "loop_fade":
		//		f, e := needFloatArg("fade", api, args)
		//		if e == nil {
		//			player.fadeLoop = f
		//		} else {
		//			err = e
		//		}
		//
		//	case "ANO":
		//		player.sendANO()

		/*
			case "midi_thru":
				v, e := needBoolArg("onoff", api, args)
				if e == nil {
					player.MIDIThru = v
				} else {
					err = e
				}

			case "midi_setscale":
				v, e := needBoolArg("onoff", api, args)
				if e == nil {
					player.MIDISetScale = v
				} else {
					err = e
				}

			case "midi_usescale":
				v, e := needBoolArg("onoff", api, args)
				if e == nil {
					player.MIDIUseScale = v
				} else {
					err = e
				}

			case "clearexternalscale":
				player.clearExternalScale()
				player.MIDINumDown = 0

			case "midi_quantized":
				v, err := needBoolArg("onoff", api, args)
				if err == nil {
					player.MIDIQuantized = v
				}

			case "midi_thruscadjust":
				v, e := needBoolArg("onoff", api, args)
				if e == nil {
					player.MIDIThruScadjust = v
				} else {
					err = e
				}

			case "set_transpose":
				v, e := needIntArg("value", api, args)
				if e == nil {
					player.TransposePitch = v
					Info("player API set_transpose", "pitch", v)
				} else {
					err = e
				}
		*/

	default:
		err = fmt.Errorf("Player.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}

func (ctx *AgentContext) sendAllParameters() {
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

func ApplyParamsMap(presetType string, paramsmap map[string]interface{}, params *ParamValues) error {

	// Currently, no errors are ever returned, but log messages are generated.

	for name, ival := range paramsmap {
		val, okval := ival.(string)
		if !okval {
			Warn("value isn't a string in params json", "name", name, "value", val)
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
		err := SetOneParamValue(params, fullname, val)
		if err != nil {
			LogError(err)
			// Don't abort the whole load, i.e. we are tolerant
			// of unknown parameters or errors in the preset
		}
	}
	return nil
}

/*
func (ctx *AgentContext) restoreCurrentSnap(layerName string) {
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

func (ctx *AgentContext) saveCurrentAsPreset(presetName string) error {
	preset, err := LoadPreset(presetName)
	if err != nil {
		return err
	}
	path := preset.WriteableFilePath()
	return ctx.saveCurrentSnapInPath(path)
}

func (ctx *AgentContext) saveCurrentSnapInPath(path string) error {

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := ctx.agentParams.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		valstring, e := ctx.agentParams.paramValueAsString(fullName)
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
