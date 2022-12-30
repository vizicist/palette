package engine

import (
	"fmt"
	"sort"
	"strings"
)

type Layer struct {
	Synth     *Synth
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
		Synth:     GetSynth("default"),
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

func ApplyToAllLayers(f func(layer *Layer)) {
	for _, layer := range Layers {
		f(layer)
	}
}

func (layer *Layer) MIDIChannel() uint8 {
	return layer.Synth.Channel()
}

func (layer *Layer) ResendAllParameters() {

	params := layer.params
	params.DoForAllParams(func(name string, val ParamValue) {
		valstr, err := params.paramValueAsString(name)
		if err != nil {
			LogError(err)
			return
		}

		// This assumes that if you set a parameter to the same value,
		// that it will re-send the mesasges to Resolume for visual.* params

		// Don't use Set, we don't want to lock
		err = layer.params.setParamValueWithString(name, valstr, nil)
		if err != nil {
			LogError(err)
		}
	})
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
		filename, okfilename := apiargs["filename"]
		if !okfilename {
			return "", fmt.Errorf("missing filename parameter")
		}
		category, okcategory := apiargs["category"]
		if !okcategory {
			return "", fmt.Errorf("missing category parameter")
		}

		err := layer.LoadSaved(category, filename)
		if err != nil {
			return "", err
		}
		layer.ResendAllParameters()
		return "", layer.SaveCurrentParams()

		/*
			category, filename := SavedNameSplit(savedName)
			path := layer.readableFilePath(category, filename)
			paramsMap, err := LoadParamsMap(path)
			if err != nil {
				return "", err
			}
			err = ApplyParamsMap(category, paramsMap, layer.params)
			if err != nil {
				return "", err
			}

			return "", err
		*/

	case "save":
		filename, okfilename := apiargs["filename"]
		if !okfilename {
			return "", fmt.Errorf("missing filename parameter")
		}
		category, okcategory := apiargs["category"]
		if !okcategory {
			return "", fmt.Errorf("missing category parameter")
		}
		// category, filename := SavedNameSplit(savedName)
		path := WritableFilePath(category, filename)
		return "", layer.params.SaveInPath(path)

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

	default:
		// ignore errors on these for the moment
		if strings.HasPrefix(api, "loop_") || strings.HasPrefix(api, "midi_") {
			return "", nil
		}
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
		args := map[string]string{
			"event": "layerset",
			"name":  paramName, "value": paramValue,
			"layer": layer.Name(),
		}
		listener.api(listener, "event", args)
	}

	return nil
}

// If no such parameter, return ""
func (layer *Layer) Get(paramName string) string {
	return layer.params.Get(paramName)
}

// func (layer *Layer) Params() *ParamValues {
// 	return layer.params
// }

func (layer *Layer) ParamNames() []string {
	// Print the parameter values sorted by name
	fullNames := layer.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		sortedNames = append(sortedNames, k)
	}
	sort.Strings(sortedNames)
	return sortedNames
}

func (layer *Layer) GetInt(paramName string) int {
	return layer.params.GetIntValue(paramName)
}

func (layer *Layer) GetFloat(paramName string) float32 {
	return layer.params.GetFloatValue(paramName)
}

func (layer *Layer) SaveCurrentParams() error {
	savedName := "layer._Current_" + layer.name
	category, filename := SavedNameSplit(savedName)
	path := WritableFilePath(category, filename)
	return layer.params.SaveInPath(path)
}

func (layer *Layer) ApplyPresetSaved(savedName string) error {
	category, filename := SavedNameSplit(savedName)
	path := layer.readableFilePath(category, filename)
	paramsMap, err := LoadParamsMap(path)
	if err != nil {
		LogError(err)
		return err
	}
	return layer.ApplyPresetSavedMap(paramsMap)
}

func (layer *Layer) ApplyPresetSavedMap(paramsmap map[string]any) error {

	for name, ival := range paramsmap {
		value, ok := ival.(string)
		if !ok {
			return fmt.Errorf("value of name=%s isn't a string", name)
		}
		// In a preset file, the parameter names are of the form:
		// {layer}-{parametername}
		words := strings.SplitN(name, "-", 2)
		layerOfParam := words[0]
		if layer.Name() != layerOfParam {
			continue
		}
		// use words[1] so the layer doesn't see the layer name
		parameterName := words[1]
		// We expect the parameter to be of the form
		// {category}.{parameter}, but old "saved" files
		// didn't include the category.
		if !strings.Contains(parameterName, ".") {
			LogWarn("applyPresetSaved: OLD format, not supported")
			return fmt.Errorf("")
		}
		err := layer.Set(parameterName, value)
		if err != nil {
			LogWarn("applyPresetSaved", "name", parameterName, "err", err)
			// Don't fail completely on individual failures,
			// some might be for parameters that no longer exist.
		}
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// saved, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing saved files.
	// This is similar to code in Layer.applySaved, except we
	// have to do it for all for pads
	for nm, def := range ParamDefs {
		paramName := layer.Name() + "-" + nm
		_, found := paramsmap[paramName]
		if !found {
			init := def.Init
			err := layer.Set(nm, init)
			if err != nil {
				// a hack to eliminate errors on a parameter that still exists somewhere
				LogWarn("applyPresetSaved", "nm", nm, "err", err)
				// Don't fail completely on individual failures,
				// some might be for parameters that no longer exist.
			}
		}
	}

	return nil
}

// ReadableSavedFilePath xxx
func (layer *Layer) readableFilePath(category string, filename string) string {
	return SavedFilePath(category, filename)
}

func (layer *Layer) LoadSaved(category string, filename string) error {

	// category, filename := SavedNameSplit(savedName)
	path := layer.readableFilePath(category, filename)
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		LogError(err)
		return err
	}
	params := layer.params
	if category == "saved" {
		// ApplyPresetSaved will only load that one layer from the preset
		err = layer.ApplyPresetSavedMap(paramsmap)
		if err != nil {
			LogError(err)
			return err
		}
	} else {
		err = ApplyParamsMap(category, paramsmap, params)
		if err != nil {
			LogError(err)
			return err
		}
	}

	// If there's a _override.json file, use it
	overrideFilename := "._override"
	overridepath := layer.readableFilePath(category, overrideFilename)
	if fileExists(overridepath) {
		DebugLogOfType("saved", "applySaved using", "overridepath", overridepath)
		overridemap, err := LoadParamsMap(overridepath)
		if err != nil {
			return err
		}
		err = ApplyParamsMap(category, overridemap, params)
		if err != nil {
			return err
		}
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// saved, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing saved files.
	for nm, def := range ParamDefs {
		// Only include parameters of the desired type
		thisCategory, _ := SavedNameSplit(nm)
		if category != "layer" && category != thisCategory {
			continue
		}
		_, found := paramsmap[nm]
		if !found {
			init := def.Init
			err := params.Set(nm, init)
			if err != nil {
				LogWarn("Loading saved", "saved", nm, "err", err)
				// Don't fail completely
			}
		}
	}
	return nil
}
