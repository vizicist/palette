package engine

type Layer struct {
	name   string
	params *ParamValues
}

var Layers = map[string]*Layer{}

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

func (layer *Layer) Set(paramName string, paramValue string) error {
	return layer.params.Set(paramName, paramValue)
}

func (layer *Layer) Apply(preset *Preset) {
	preset.ApplyTo(layer.params)
}
