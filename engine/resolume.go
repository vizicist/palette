package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var ResolumePort = 7000

type Resolume struct {
	resolumeClient   *osc.Client
	freeframeClients map[string]*osc.Client
}

var theResolume *Resolume

func TheResolume() *Resolume {
	if theResolume == nil {
		theResolume = &Resolume{
			resolumeClient:   osc.NewClient(LocalAddress, ResolumePort),
			freeframeClients: map[string]*osc.Client{},
		}

		_ = theResolume.bypassLayer // to avoid unused error

		err := theResolume.loadResolumeJSON()
		if err != nil {
			LogError(err)
		}
	}
	return theResolume
}

// ResolumeJSON is an unmarshalled version of the resolume.json file
var ResolumeJSON map[string]any

// LoadResolumeJSON returns an unmarshalled version of the resolume.json file
func (r *Resolume) loadResolumeJSON() error {
	path := ConfigFilePath("resolume.json")
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

func (r *Resolume) PortAndLayerNumForPatch(patchName string) (portNum, layerNum int) {
	switch patchName {
	case "A":
		return 3334, 1
	case "B":
		return 3335, 2
	case "C":
		return 3336, 3
	case "D":
		return 3337, 4
	default:
		LogError(fmt.Errorf("no port for layer %s", patchName))
		return 0, 0
	}
}

func (r *Resolume) freeframeClientFor(patchName string) *osc.Client {
	ff, ok := r.freeframeClients[patchName]
	if !ok {
		portNum, _ := r.PortAndLayerNumForPatch(patchName)
		if portNum == 0 {
			return nil
		}
		ff = osc.NewClient(LocalAddress, portNum)
		r.freeframeClients[patchName] = ff
	}
	return ff
}

func (r *Resolume) ToFreeFramePlugin(patchName string, msg *osc.Message) {
	LogOfType("freeframe", "Resolume.toFreeframe", "patch", patchName, "msg", msg)
	ff := r.freeframeClientFor(patchName)
	if ff == nil {
		LogError(fmt.Errorf("no freeframe client for layer"), "patch", patchName)
		return
	}
	LogError(ff.Send(msg))
}

func (r *Resolume) SendEffectParam(patchName string, name string, value string) {
	portNum, layerNum := r.PortAndLayerNumForPatch(patchName)
	if portNum == 0 {
		LogError(fmt.Errorf("no such layer"), "name", patchName)
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
			LogError(err)
			onoff = false
		}
		r.sendPadOneEffectOnOff(layerNum, name, onoff)
	}
}

func (r *Resolume) sendPadOneEffectParam(layerNum int, effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
	paramsMap, realEffectName, realEffectNum, err := r.getEffectMap(effectName, "params")
	if err != nil {
		LogError(err)
		return
	}
	if paramsMap == nil {
		LogWarn("No params value for", "effecdt", effectName)
		return
	}
	oneParam, ok := paramsMap[paramName]
	if !ok {
		LogWarn("No params value for", "param", paramName, "effect", effectName)
		return
	}

	oneDef, ok := ParamDefs[fullName]
	if !ok {
		LogWarn("No paramdef value for", "param", paramName, "effect", effectName)
		return
	}

	addr := oneParam.(string)
	resEffectName := resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)
	addr = addLayerAndClipNums(addr, layerNum, 1)

	msg := osc.NewMessage(addr)

	// Append the value to the message, depending on the type of the parameter

	switch oneDef.TypedParamDef.(type) {

	case ParamDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			LogError(err)
			valint = 0
		}
		msg.Append(int32(valint))

	case ParamDefBool:
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

	case ParamDefString:
		valstr := value
		msg.Append(valstr)

	case ParamDefFloat:
		var valfloat float32
		valfloat, err := ParseFloat32(value, resEffectName)
		if err != nil {
			LogError(err)
			valfloat = 0.0
		}
		msg.Append(float32(valfloat))

	default:
		LogWarn("SetParamValueWithString: unknown type of ParamDef for", "name", fullName)
		return
	}

	r.toResolume(msg)
}

func (r *Resolume) toResolume(msg *osc.Message) {
	LogOfType("resolume", "Resolume.toResolume", "msg", msg)
	LogError(r.resolumeClient.Send(msg))
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
		LogError(err)
		return
	}

	if onoffMap == nil {
		LogWarn("No onoffMap value for", "effect", effectName, "maptype", mapType, effectName)
		return
	}

	onoffAddr, ok := onoffMap["addr"]
	if !ok {
		LogWarn("No addr value in onoff", "effect", effectName)
		return
	}
	onoffArg, ok := onoffMap["arg"]
	if !ok {
		LogWarn("No arg valuei in onoff for", "effect", effectName)
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

	// make sure the layer is not displayed before changing it
	r.bypassLayer(textLayerNum, true)

	// the first clip is the textgenerator clip
	addr := fmt.Sprintf("/composition/layers/%d/clips/1/video/source/textgenerator/text/params/lines", textLayerNum)
	msg := osc.NewMessage(addr)
	msg.Append(text)
	_ = r.resolumeClient.Send(msg)

	// give it time to "sink in", otherwise the previous text displays briefly
	time.Sleep(150 * time.Millisecond)

	r.connectClip(textLayerNum, 1)     // activate that clip
	r.bypassLayer(textLayerNum, false) // show the layer
}

func (r *Resolume) ResolumeLayerForText() int {
	defLayer := "5"
	s := TheEngine.GetWithDefault("textlayer", defLayer)
	layernum, err := strconv.Atoi(s)
	if err != nil {
		LogError(err)
		layernum, _ = strconv.Atoi(defLayer)
	}
	return layernum
}

func (r *Resolume) ProcessInfo() *ProcessInfo {
	fullpath := ConfigValue("resolume")
	if fullpath != "" && !FileExists(fullpath) {
		LogWarn("No Resolume found, looking for", "path", fullpath)
		return nil
	}
	if fullpath == "" {
		fullpath = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
		if !FileExists(fullpath) {
			fullpath = "C:\\Program Files\\Resolume Arena\\Arena.exe"
			if !FileExists(fullpath) {
				LogWarn("Resolume not found in default locations")
				return nil
			}
		}
	}
	exe := fullpath
	lastslash := strings.LastIndex(fullpath, "\\")
	if lastslash > 0 {
		exe = fullpath[lastslash+1:]
	}
	return NewProcessInfo(exe, fullpath, "", r.Activate)
}

func (r *Resolume) Activate() {
	// handle_activate sends OSC messages to start the layers in Resolume,
	textLayer := r.ResolumeLayerForText()
	clipnum := 1

	// do it a few times, in case Resolume hasn't started up
	for i := 0; i < 4; i++ {
		time.Sleep(5 * time.Second)

		for _, patch := range PatchNames() {
			_, layerNum := r.PortAndLayerNumForPatch(string(patch))
			LogOfType("resolume", "Activating Resolume", "patch", layerNum, "clipnum", clipnum)
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