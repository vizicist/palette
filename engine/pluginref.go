package engine

type PluginRef struct {
	name   string
	active bool
}

func NewPluginRef(name string) *PluginRef {
	return &PluginRef{
		name:   name,
		active: false,
	}
}
