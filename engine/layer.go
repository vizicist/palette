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
	layer.SetDefaultValues()
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

func IsPerLayerParam(name string) bool {
	// Layer params are sound.*, visual.* and effect.*
	if strings.HasPrefix(name, "misc.") || strings.HasPrefix(name, "global.") {
		return false
	}
	return true
}

func IsPerPresetParam(name string) bool {
	// Preset params are everything but global.*,
	// i.e. misc.*, sound.*, visual.* and effect.*
	if strings.HasPrefix(name, "global.") {
		return false
	}
	return true
}

func ApplyToAllLayers(f func(layer *Layer)) {
	for _, layer := range Layers {
		f(layer)
	}
}

// SetDefaultValues xxx
func (layer *Layer) SetDefaultValues() {
	for nm, d := range ParamDefs {
		if IsPerLayerParam(nm) {
			err := layer.Set(nm, d.Init)
			// err := layer.params.SetParamValueWithString(nm, d.Init)
			if err != nil {
				LogError(err)
			}
		}
	}
}

func (layer *Layer) MIDIChannel() uint8 {
	return layer.Synth.Channel()
}

func (layer *Layer) ReAlertAllParameters() {

	params := layer.params
	params.DoForAllParams(func(paramName string, paramValue ParamValue) {
		if !IsPerLayerParam(paramName) {
			LogWarn("ResendAllParameters: skipping param %s", paramName)
			return
		}
		valstr, err := params.paramValueAsString(paramName)
		if err != nil {
			LogError(err)
			return
		}
		layer.alertListeners(paramName, valstr)
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

		err := layer.Load(category, filename)
		if err != nil {
			return "", err
		}
		layer.ReAlertAllParameters()
		return "", layer.SaveCurrent()

		/*
			category, filename := SavedNameSplit(savedName)
			path := layer.readableFilePath(category, filename)
			paramsMap, err := LoadParamsMap(path)
			if err != nil {
				return "", err
			}
			err = layer.parms.ApplyParamsMap(category, paramsMap)
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
		return "", layer.params.Save(category, filename)

	case "set":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", fmt.Errorf("executeLayerAPI: err=%s", err)
		}
		layer.Set(name, value)
		return "", layer.SaveCurrent()

	case "setparams":
		for name, value := range apiargs {
			layer.Set(name, value)
		}
		return "", layer.SaveCurrent()

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

func (layer *Layer) Set(paramName string, paramValue string) error {

	if !IsPerLayerParam(paramName) {
		return fmt.Errorf("Layer.Set: not a per-layer param")
	}
	err := layer.params.Set(paramName, paramValue)
	if err != nil {
		return err
	}
	LogInfo("Layer.Set: alerting listeners for %s=%s", paramName, paramValue)
	layer.alertListeners(paramName, paramValue)
	return nil
}

func (layer *Layer) alertListeners(paramName string, paramValue string) {

	for _, listener := range layer.listeners {
		args := map[string]string{
			"event": "layerset",
			"name":  paramName,
			"value": paramValue,
			"layer": layer.Name(),
		}
		listener.api(listener, "event", args)
	}
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
		if IsPerLayerParam(k) {
			sortedNames = append(sortedNames, k)
		}
	}
	sort.Strings(sortedNames)
	return sortedNames
}

func (layer *Layer) GetString(paramName string) string {
	return layer.params.GetStringValue(paramName, "")
}

func (layer *Layer) GetInt(paramName string) int {
	return layer.params.GetIntValue(paramName)
}

func (layer *Layer) GetBool(paramName string) bool {
	return layer.params.GetBoolValue(paramName)
}

func (layer *Layer) GetFloat(paramName string) float32 {
	return layer.params.GetFloatValue(paramName)
}

func (layer *Layer) SaveCurrent() error {
	return layer.params.Save("layer", "_Current")
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
		if !IsPerLayerParam(name) {
			LogWarn("ApplyPresetSavedMap: phase 1 ignoring non-layer param", "name", name)
		}
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
	for paramName, def := range ParamDefs {
		if !IsPerLayerParam(paramName) {
			continue
		}
		layerParamName := layer.Name() + "-" + paramName
		_, found := paramsmap[layerParamName]
		if !found {
			init := def.Init
			err := layer.Set(paramName, init)
			if err != nil {
				// a hack to eliminate errors on a parameter that still exists somewhere
				LogWarn("applyPresetSaved", "nm", paramName, "err", err)
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

func (layer *Layer) Load(category string, filename string) error {

	// category, filename := SavedNameSplit(savedName)
	path := layer.readableFilePath(category, filename)
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		LogError(err)
		return err
	}
	params := layer.params
	if category == "preset" {
		// ApplyPresetSaved will only load that one layer from the preset
		err = layer.ApplyPresetSavedMap(paramsmap)
		if err != nil {
			LogError(err)
			return err
		}
	} else {
		params.ApplyParamsMap(category, paramsmap)
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
		params.ApplyParamsMap(category, overridemap)
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// saved, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing saved files.
	for paramName, def := range ParamDefs {
		// Only include parameters of the desired type
		thisCategory, _ := SavedNameSplit(paramName)
		if category != "layer" && category != thisCategory {
			continue
		}
		_, found := paramsmap[paramName]
		if !found {
			paramValue := def.Init
			err := layer.Set(paramName, paramValue)
			// err := params.Set(paramName, paramValue)
			if err != nil {
				LogWarn("Loading saved", "saved", paramName, "err", err)
				// Don't fail completely
			}
		}
	}
	layer.ReAlertAllParameters()
	return nil
}
