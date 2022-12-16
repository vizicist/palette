package engine

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Resolume struct {
	resolumeClient   *osc.Client
	freeframeClients map[string]*osc.Client
}

var oneResolume *Resolume

func TheResolume() *Resolume {
	if oneResolume == nil {
		oneResolume = NewResolume()
	}
	return oneResolume
}

func NewResolume() *Resolume {
	return &Resolume{
		resolumeClient:   osc.NewClient(LocalAddress, ResolumePort),
		freeframeClients: map[string]*osc.Client{},
	}
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
		LogError(fmt.Errorf("no port for layer %s", layerName))
		return 0, 0
	}
}

func (r *Resolume) clientFor(layerName string) *osc.Client {
	ff, ok := r.freeframeClients[layerName]
	if !ok {
		portNum, _ := r.infoForLayer(layerName)
		if portNum == 0 {
			LogError(fmt.Errorf("Resolume.clientFor: no info for layerName"), "name", layerName)
		}
		ff = osc.NewClient(LocalAddress, portNum)
		r.freeframeClients[layerName] = ff
	}
	return ff
}

func (r *Resolume) toFreeFramePlugin(layerName string, msg *osc.Message) {
	ff, ok := r.freeframeClients[layerName]
	if !ok {
		LogError(fmt.Errorf("no freeframe client"), "name", layerName)
		return
	}
	ff.Send(msg)
}

func (r *Resolume) sendEffectParam(layerName string, name string, value string) {
	portNum, layerNum := r.infoForLayer(layerName)
	if portNum == 0 {
		LogError(fmt.Errorf("no such layer"), "name", layerName)
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
	paramsMap, realEffectName, realEffectNum, err := getEffectMap(effectName, "params")
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
		LogWarn("SetParamValueWithString: unknown type of ParamDef for", "name", fullName)
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

	onoffMap, realEffectName, realEffectNum, err := getEffectMap(effectName, mapType)
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
	s := ConfigStringWithDefault("textlayer", defLayer)
	layernum, err := strconv.Atoi(s)
	if err != nil {
		LogError(err)
		layernum, _ = strconv.Atoi(defLayer)
	}
	return layernum
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
			DebugLogOfType("resolume", "Activating Resolume", "layer", layerNum, "clipnum", clipnum)
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
