package engine

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type Layer struct {
	name   string
	params *ParamValues
}

var Layers = map[string]*Layer{}

func NewLayer(layerName string) *Layer {
	layer := &Layer{
		name:   layerName,
		params: NewParamValues(),
	}
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
			err = layer.ApplyQuadPreset(preset, layer.Name())
			if err != nil {
				LogError(err)
				return "", err
			}
			layer.SaveCurrentSnap()
		} else {
			// It's a non-quad preset for a single layer.
			layer.Apply(preset)
			err := layer.SaveCurrentSnap()
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
		return "", layer.SaveCurrentSnap()

	case "setparams":
		for name, value := range apiargs {
			layer.Set(name, value)
		}
		return "", layer.SaveCurrentSnap()

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executeLayerAPI: missing name argument")
		}
		return layer.Get(name), nil

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

	resolume := TheResolume()
	layerName := layer.name

	if strings.HasPrefix(paramName, "visual.") {
		name := strings.TrimPrefix(paramName, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, paramValue)
		msg.Append(args)
		resolume.toFreeFramePlugin(layerName,msg)
	}

	if strings.HasPrefix(paramName, "effect.") {
		name := strings.TrimPrefix(paramName, "effect.")
		// Effect parameters get sent to Resolume
		resolume.sendEffectParam(layerName, name, paramValue)
	}

	return nil
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
	for _, c := range TheRouter().layerLetters {
		layerName := string(c)
		for nm, def := range ParamDefs {
			paramName := string(layerName) + "-" + nm
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

	TheResolume().toFreeFramePlugin(layer.name, msg)
}
