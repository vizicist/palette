package engine

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type Layer struct {
	// name   string
	params          *ParamValues
	freeframeClient *osc.Client
	resolumeClient  *osc.Client
	resolumeLayer   int // see ResolumeLayerForPad
}

var Layers = map[string]*Layer{}

func GetLayer(layerName string) *Layer {
	layer, ok := Layers[layerName]
	if !ok {
		layer = &Layer{params: NewParamValues()}
		Layers[layerName] = layer
	}
	return layer
}

func (layer *Layer) toFreeFramePlugin(msg *osc.Message) {
	if layer.freeframeClient == nil {
		ffglPort := layer.GetInt("visual.ffglport")
		layer.freeframeClient = osc.NewClient(LocalAddress, ffglPort)
	}
	layer.freeframeClient.Send(msg)
}

func (layer *Layer) sendEffectParam(name string, value string) {
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		layer.sendPadOneEffectParam(effectName, paramName, value)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			LogError(err)
			onoff = false
		}
		layer.sendPadOneEffectOnOff(name, onoff)
	}
}

func (layer *Layer) sendPadOneEffectParam(effectName string, paramName string, value string) {
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

	layer.toResolume(msg)
}

func (layer *Layer) sendPadOneEffectOnOff(effectName string, onoff bool) {
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
	addr = layer.addEffectNum(addr, realEffectName, realEffectNum)
	addr = addLayerAndClipNums(addr, layer.resolumeLayer, 1)
	onoffValue := int(onoffArg.(float64))

	msg := osc.NewMessage(addr)
	msg.Append(int32(onoffValue))
	layer.toResolume(msg)
}

func (layer *Layer) addEffectNum(addr string, effect string, num int) string {
	if num == 1 {
		return addr
	}
	// e.g. "blur" becomes "blur2"
	return strings.Replace(addr, effect, fmt.Sprintf("%s%d", effect, num), 1)
}

/*
func MakeLayer(name string) *Layer {
	layer, ok := Layers[name]
	if !ok {
		layer = &Layer{name: name, params: NewParamValues()}
		Layers[name] = layer
	} else {
		Warn("MakeLayer: layer already exists?", "layer", layer)
	}
	return layer
}
*/

func (layer *Layer) Set(paramName string, paramValue string) error {
	return layer.params.Set(paramName, paramValue)
}

// If no such parameter, return ""
func (layer *Layer) Get(paramName string) string {
	return layer.params.Get(paramName)
}

// If no such parameter, return ""
func (layer *Layer) GetInt(paramName string) int {
	return layer.params.ParamIntValue(paramName)
}

func (layer *Layer) Apply(preset *Preset) {
	preset.ApplyTo(layer.params)
}

func (layer *Layer) SaveCurrentSnap() error {
	return fmt.Errorf("Layer.SaveCurrentSnap needs work")
}

func (layer *Layer) SaveCurrentPreset(path string) error {

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := layer.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		valstring, e := layer.params.paramValueAsString(fullName)
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

func (layer *Layer) ApplyQuadPreset(preset *Preset, layerToApply string) error {
	// Here's where the params get applied,
	// which among other things
	// may result in sending OSC messages out.
	for name, ival := range preset.paramsmap {
		value, ok := ival.(string)
		if !ok {
			return fmt.Errorf("value of name=%s isn't a string", name)
		}
		// In a quad file, the parameter names are of the form:
		// {layer}-{parametername}
		words := strings.SplitN(name, "-", 2)
		layerOfParam := words[0]
		if layerToApply != layerOfParam {
			continue
		}
		// use words[1] so the layer doesn't see the layer name
		parameterName := words[1]
		// We expect the parameter to be of the form
		// {category}.{parameter}, but old "quad" files
		// didn't include the category.
		if !strings.Contains(parameterName, ".") {
			Warn("applyQuadPreset: OLD format, not supported")
			return fmt.Errorf("")
		}
		err := layer.Set(parameterName, value)
		if err != nil {
			Warn("applyQuadPreset", "name", parameterName, "err", err)
			// Don't fail completely on individual failures,
			// some might be for parameters that no longer exist.
		}
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// preset, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing preset files.
	// This is similar to code in Layer.applyPreset, except we
	// have to do it for all for pads
	for _, c := range TheRouter().playerLetters {
		playerName := string(c)
		for nm, def := range ParamDefs {
			paramName := string(playerName) + "-" + nm
			_, found := preset.paramsmap[paramName]
			if !found {
				init := def.Init
				err := layer.Set(nm, init)
				if err != nil {
					// a hack to eliminate errors on a parameter that
					// still exists in some presets.
					Warn("applyQuadPreset", "nm", nm, "err", err)
					// Don't fail completely on individual failures,
					// some might be for parameters that no longer exist.
				}
			}
		}
	}

	return nil
}

/*
func (layer *Layer) ffglPort() int {
	layers := "ABCDEFGH"
	i := strings.Index(layers, layerName)
	return 3334 + i
}
*/

func (layer *Layer) toResolume(msg *osc.Message) {
	if layer.resolumeClient == nil {
		layer.resolumeClient = osc.NewClient(LocalAddress, ResolumePort)
	}
	layer.resolumeClient.Send(msg)
}

func (layer *Layer) generateSprite(id string, x, y, z float32) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	layer.toFreeFramePlugin(msg)
}
