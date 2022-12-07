package engine

type Layer struct {
	// name   string
	params *ParamValues
}

func NewLayer() *Layer {
	return &Layer{params: NewParamValues()}
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
