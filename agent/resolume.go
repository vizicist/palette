package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

var ResolumePort = 7000

type Resolume struct {
	agent            *engine.AgentContext
	resolumeClient   *osc.Client
	freeframeClients map[string]*osc.Client
}

// ResolumeJSON is an unmarshalled version of the resolume.json file
var ResolumeJSON map[string]any

func NewResolume(agent *engine.AgentContext) *Resolume {
	r := &Resolume{
		agent:            agent,
		resolumeClient:   osc.NewClient(engine.LocalAddress, ResolumePort),
		freeframeClients: map[string]*osc.Client{},
	}
	if err := r.loadResolumeJSON(); err != nil {
		engine.LogError(err)
	}
	return r
}

// LoadResolumeJSON returns an unmarshalled version of the resolume.json file
func (r *Resolume) loadResolumeJSON() error {
	path := engine.ConfigFilePath("resolume.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read resolume.json, err=%s", err)
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s", path)
	}
	ResolumeJSON = f.(map[string]any)
	return nil
}

func (r *Resolume) infoForLayer(layerName string) (portNum, layerNum int) {
	switch layerName {
	case "a":
		return 3334, 1
	case "b":
		return 3335, 2
	case "c":
		return 3336, 3
	case "d":
		return 3337, 4
	default:
		engine.LogError(fmt.Errorf("no port for layer %s", layerName))
		return 0, 0
	}
}

func (r *Resolume) freeframeClientFor(layerName string) *osc.Client {
	ff, ok := r.freeframeClients[layerName]
	if !ok {
		portNum, _ := r.infoForLayer(layerName)
		if portNum == 0 {
			return nil
		}
		ff = osc.NewClient(engine.LocalAddress, portNum)
		r.freeframeClients[layerName] = ff
	}
	return ff
}

func (r *Resolume) toFreeFramePlugin(layerName string, msg *osc.Message) {
	engine.DebugLogOfType("freeframe", "toFreeframe", "layer", layerName, "msg", msg)
	ff := r.freeframeClientFor(layerName)
	if ff == nil {
		engine.LogError(fmt.Errorf("no freeframe client for layer"), "layer", layerName)
		return
	}
	ff.Send(msg)
}

func (r *Resolume) sendEffectParam(layerName string, name string, value string) {
	portNum, layerNum := r.infoForLayer(layerName)
	if portNum == 0 {
		engine.LogError(fmt.Errorf("no such layer"), "name", layerName)
		return
	}
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		r.sendPadOneEffectParam(layerNum, effectName, paramName, value)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			engine.LogError(err)
			onoff = false
		}
		r.sendPadOneEffectOnOff(layerNum, name, onoff)
	}
}

func (r *Resolume) sendPadOneEffectParam(layerNum int, effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
	paramsMap, realEffectName, realEffectNum, err := r.getEffectMap(effectName, "params")
	if err != nil {
		engine.LogError(err)
		return
	}
	if paramsMap == nil {
		engine.LogWarn("No params value for", "effecdt", effectName)
		return
	}
	oneParam, ok := paramsMap[paramName]
	if !ok {
		engine.LogWarn("No params value for", "param", paramName, "effect", effectName)
		return
	}

	oneDef, ok := engine.ParamDefs[fullName]
	if !ok {
		engine.LogWarn("No paramdef value for", "param", paramName, "effect", effectName)
		return
	}

	addr := oneParam.(string)
	resEffectName := resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)
	addr = addLayerAndClipNums(addr, layerNum, 1)

	msg := osc.NewMessage(addr)

	// Append the value to the message, depending on the type of the parameter

	switch oneDef.TypedParamDef.(type) {

	case engine.ParamDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			engine.LogError(err)
			valint = 0
		}
		msg.Append(int32(valint))

	case engine.ParamDefBool:
		valbool, err := strconv.ParseBool(value)
		if err != nil {
			engine.LogError(err)
			valbool = false
		}
		onoffValue := 0
		if valbool {
			onoffValue = 1
		}
		msg.Append(int32(onoffValue))

	case engine.ParamDefString:
		valstr := value
		msg.Append(valstr)

	case engine.ParamDefFloat:
		var valfloat float32
		valfloat, err := engine.ParseFloat32(value, resEffectName)
		if err != nil {
			engine.LogError(err)
			valfloat = 0.0
		}
		msg.Append(float32(valfloat))

	default:
		engine.LogWarn("SetParamValueWithString: unknown type of ParamDef for", "name", fullName)
		return
	}

	r.toResolume(msg)
}

func (r *Resolume) toResolume(msg *osc.Message) {
	r.resolumeClient.Send(msg)
}

func (r *Resolume) sendPadOneEffectOnOff(layerNum int, effectName string, onoff bool) {
	var mapType string
	if onoff {
		mapType = "on"
	} else {
		mapType = "off"
	}

	onoffMap, realEffectName, realEffectNum, err := r.getEffectMap(effectName, mapType)
	if err != nil {
		engine.LogError(err)
		return
	}

	if onoffMap == nil {
		engine.LogWarn("No onoffMap value for", "effect", effectName, "maptype", mapType, effectName)
		return
	}

	onoffAddr, ok := onoffMap["addr"]
	if !ok {
		engine.LogWarn("No addr value in onoff", "effect", effectName)
		return
	}
	onoffArg, ok := onoffMap["arg"]
	if !ok {
		engine.LogWarn("No arg valuei in onoff for", "effect", effectName)
		return
	}
	addr := onoffAddr.(string)
	addr = r.addEffectNum(addr, realEffectName, realEffectNum)
	addr = addLayerAndClipNums(addr, layerNum, 1)
	onoffValue := int(onoffArg.(float64))

	msg := osc.NewMessage(addr)
	msg.Append(int32(onoffValue))
	r.toResolume(msg)
}

func (r *Resolume) addEffectNum(addr string, effect string, num int) string {
	if num == 1 {
		return addr
	}
	// e.g. "blur" becomes "blur2"
	return strings.Replace(addr, effect, fmt.Sprintf("%s%d", effect, num), 1)
}

func (r *Resolume) showText(text string) {

	textLayerNum := r.ResolumeLayerForText()

	// disable the text display by bypassing the layer
	r.bypassLayer(textLayerNum, true)

	addr := fmt.Sprintf("/composition/layers/%d/clips/1/video/source/textgenerator/text/params/lines", textLayerNum)
	msg := osc.NewMessage(addr)
	msg.Append(text)
	_ = r.resolumeClient.Send(msg)

	// give it time to "sink in", otherwise the previous text displays briefly
	time.Sleep(150 * time.Millisecond)

	r.connectClip(textLayerNum, 1)
	r.bypassLayer(textLayerNum, false)
}

func (r *Resolume) ResolumeLayerForText() int {
	defLayer := "5"
	s := engine.ConfigStringWithDefault("textlayer", defLayer)
	layernum, err := strconv.Atoi(s)
	if err != nil {
		engine.LogError(err)
		layernum, _ = strconv.Atoi(defLayer)
	}
	return layernum
}

func (r *Resolume) ProcessInfo() *processInfo {
	fullpath := engine.ConfigValue("resolume")
	if fullpath != "" && !r.agent.FileExists(fullpath) {
		engine.LogWarn("No Resolume found, looking for", "path", fullpath)
		return nil
	}
	if fullpath == "" {
		fullpath = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
		if !r.agent.FileExists(fullpath) {
			fullpath = "C:\\Program Files\\Resolume Arena\\Arena.exe"
			if !r.agent.FileExists(fullpath) {
				engine.LogWarn("Resolume not found in default locations")
				return nil
			}
		}
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	return &processInfo{exe, fullpath, "", r.Activate}
}

func (r *Resolume) Activate() {
	// handle_activate sends OSC messages to start the layers in Resolume,
	textLayer := r.ResolumeLayerForText()
	clipnum := 1

	// do it a few times, in case Resolume hasn't started up
	for i := 0; i < 4; i++ {
		time.Sleep(5 * time.Second)

		// XXX - this assumes there are 4 layers named a,b,c,d
		// XXX - shouldn't be hardcoded here
		layerNames := []string{"a", "b", "c", "d"}

		for _, pad := range layerNames {
			_, layerNum := r.infoForLayer(string(pad))
			engine.DebugLogOfType("resolume", "Activating Resolume", "layer", layerNum, "clipnum", clipnum)
			r.connectClip(layerNum, clipnum)
		}
		if textLayer >= 1 {
			r.connectClip(textLayer, clipnum)
		}
	}
}

func (r *Resolume) connectClip(layerNum int, clip int) {
	addr := fmt.Sprintf("/composition/layers/%d/clips/%d/connect", layerNum, clip)
	msg := osc.NewMessage(addr)
	// Note: sending 0 doesn't seem to disable a clip; you need to
	// bypass the layer to turn it off
	msg.Append(int32(1))
	_ = r.resolumeClient.Send(msg)
}

func (r *Resolume) bypassLayer(layerNum int, onoff bool) {
	addr := fmt.Sprintf("/composition/layers/%d/bypassed", layerNum)
	msg := osc.NewMessage(addr)
	v := 0
	if onoff {
		v = 1
	}
	msg.Append(int32(v))
	_ = r.resolumeClient.Send(msg)
}

// getEffectMap returns the resolume.json map for a given effect
// and map type ("on", "off", or "params")
func (r *Resolume) getEffectMap(effectName string, mapType string) (map[string]any, string, int, error) {
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
		engine.LogWarn("addr in resolume.json doesn't start with /", "addr", addr)
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

	switch oneDef.TypedParamDef.(type) {

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
