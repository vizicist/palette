package engine

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type Layer struct {
	name      string
	params    *ParamValues
	listeners []*PluginContext
}

// type LayerCallbackMethods interface {
// 	OnParamSet(layer *Layer, paramName string, paramValue string)
// 	OnSpriteGen(layer *Layer, id string, x, y, z float32)
// }

var Layers = map[string]*Layer{}

func LayerNames() []string {
	arr := []string{}
	for nm := range Layers {
		arr = append(arr, nm)
	}
	return arr
}

func NewLayer(layerName string) *Layer {
	layer := &Layer{
		name:      layerName,
		params:    NewParamValues(),
		listeners: []*PluginContext{},
	}
	DebugLogOfType("layer", "NewLayer", "layer", layerName)
	layer.params.SetDefaultValues()
	Layers[layerName] = layer
	return layer
}

func GetLayer(layerName string) *Layer {
	layer, ok := Layers[layerName]
	if !ok {
		return nil
	}
	return layer
}

func (layer *Layer) AddListener(ctx *PluginContext) {
	layer.listeners = append(layer.listeners, ctx)
}

func (layer *Layer) Name() string {
	return layer.name
}

func (layer *Layer) Api(api string, apiargs map[string]string) (string, error) {

	switch api {

	case "load":
		presetName, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		preset := GetPreset(presetName)
		err := preset.LoadPreset()
		if err != nil {
			return "", err
		}

		if preset.Category == "quad" {
			// ApplyQuadPreset will only load that one layer from the quad preset
			err = layer.ApplyQuadPreset(preset)
			if err != nil {
				LogError(err)
				return "", err
			}
			layer.SaveCurrentParams()
		} else {
			// It's a non-quad preset for a single layer.
			layer.Apply(preset)
			err := layer.SaveCurrentParams()
			if err != nil {
				return "", err
			}
		}
		return "", err

	case "save":
		presetName, okpreset := apiargs["preset"]
		if !okpreset {
			return "", fmt.Errorf("missing preset parameter")
		}
		preset := GetPreset(presetName)
		path := preset.WritableFilePath()
		return "", layer.SavePresetInPath(path)

	case "set":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeLayerAPI: missing name argument")
		}
		value, ok := apiargs["value"]
		if !ok {
			return "", fmt.Errorf("executeLayerAPI: missing value argument")
		}
		layer.Set(name, value)
		return "", layer.SaveCurrentParams()

	case "setparams":
		for name, value := range apiargs {
			layer.Set(name, value)
		}
		return "", layer.SaveCurrentParams()

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeLayerAPI: missing name argument")
		}
		return layer.Get(name), nil

	case "list":
		return layer.List(), nil

	default:
		err := fmt.Errorf("Layer.API: unrecognized api=%s", api)
		LogError(err)
		return "", err
	}
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

	err := layer.params.Set(paramName, paramValue)
	if err != nil {
		return err
	}

	for _, listener := range layer.listeners {
		args := map[string]string{"name": paramName, "value": paramValue}
		listener.api(listener, "onparamset", args)
	}

	return nil
}

// If no such parameter, return ""
func (layer *Layer) Get(paramName string) string {
	return layer.params.Get(paramName)
}

func (layer *Layer) List() string {
	return layer.params.List()
}

func (layer *Layer) GetInt(paramName string) int {
	return layer.params.ParamIntValue(paramName)
}

func (layer *Layer) GetFloat(paramName string) float32 {
	return layer.params.ParamFloatValue(paramName)
}

func (layer *Layer) Apply(preset *Preset) {
	preset.ApplyTo(layer.params)
}

func (layer *Layer) SaveCurrentParams() error {
	return fmt.Errorf("Layer.SaveCurrentParams needs work")
}

func (layer *Layer) SavePresetInPath(path string) error {

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

func (layer *Layer) ApplyQuadPreset(preset *Preset) error {
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
		if layer.Name() != layerOfParam {
			continue
		}
		// use words[1] so the layer doesn't see the layer name
		parameterName := words[1]
		// We expect the parameter to be of the form
		// {category}.{parameter}, but old "quad" files
		// didn't include the category.
		if !strings.Contains(parameterName, ".") {
			LogWarn("applyQuadPreset: OLD format, not supported")
			return fmt.Errorf("")
		}
		err := layer.Set(parameterName, value)
		if err != nil {
			LogWarn("applyQuadPreset", "name", parameterName, "err", err)
			// Don't fail completely on individual failures,
			// some might be for parameters that no longer exist.
		}
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// preset, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing preset files.
	// This is similar to code in Layer.applyPreset, except we
	// have to do it for all for pads
	for nm, def := range ParamDefs {
		paramName := layer.Name() + "-" + nm
		_, found := preset.paramsmap[paramName]
		if !found {
			init := def.Init
			err := layer.Set(nm, init)
			if err != nil {
				// a hack to eliminate errors on a parameter that
				// still exists in some presets.
				LogWarn("applyQuadPreset", "nm", nm, "err", err)
				// Don't fail completely on individual failures,
				// some might be for parameters that no longer exist.
			}
		}
	}

	return nil
}
