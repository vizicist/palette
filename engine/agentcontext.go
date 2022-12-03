package engine

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/hypebeast/go-osc/osc"
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
		scheduler: TheEngine().Scheduler,
		agent:     agent,
		// params:    NewParamValues(),
		scale:   &Scale{},
		sources: map[string]bool{},
	}
}

func (ctx *AgentContext) Log(msg string, keysAndValues ...interface{}) {
	Info(msg, keysAndValues...)
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

	bytes := me.msg.Bytes()
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

func (ctx *AgentContext) SetParam(fullname, value string) error {
	return ctx.SetOneParamValue(fullname, value)
}

func (ctx *AgentContext) SetOneParamValue(fullname, value string) error {

	err := ctx.agentParams.SetParamValueWithString(fullname, value, nil)
	if err != nil {
		return err
	}

	if strings.HasPrefix(fullname, "visual.") {
		name := strings.TrimPrefix(fullname, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, value)
		msg.Append(args)
		ctx.toFreeFramePluginForLayer(msg)
	}

	if strings.HasPrefix(fullname, "effect.") {
		name := strings.TrimPrefix(fullname, "effect.")
		// Effect parameters get sent to Resolume
		ctx.sendEffectParam(name, value)
	}

	return nil
}

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

func (ctx *AgentContext) sendPadOneEffectParam(effectName string, paramName string, value string) {
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

func (ctx *AgentContext) toFreeFramePluginForLayer(msg *osc.Message) {
	ctx.freeframeClient.Send(msg)
}

func (ctx *AgentContext) toResolume(msg *osc.Message) {
	ctx.resolumeClient.Send(msg)
}

func (ctx *AgentContext) generateSprite(id string, x, y, z float32) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	ctx.toFreeFramePluginForLayer(msg)
}

func (ctx *AgentContext) generateSpriteFromPhraseElement(pe *PhraseElement) {

	var pitch uint8
	var velocity uint8

	switch v := pe.Value.(type) {
	case *NoteOn:
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteOff:
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteFull:
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

	ctx.toFreeFramePluginForLayer(msg)
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

func (ctx *AgentContext) sendPadOneEffectOnOff(effectName string, onoff bool) {
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

func (ctx *AgentContext) AllowSource(source string) {
	var ok bool
	_, ok = ctx.sources[source]
	if ok {
		Info("AllowSource already set", "source", source)
	} else {
		ctx.sources[source] = true
	}
}

func (ctx *AgentContext) ApplyPreset(presetName string) error {
	return fmt.Errorf("ApplyPreset needs work")
	/*
		preset, err := LoadPreset(presetName)
		if err != nil {
			return err
		}
		// preset.ApplyTo(p.playerName)
		return nil
	*/
}
