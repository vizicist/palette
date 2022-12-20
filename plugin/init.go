package plugin

import (
	"github.com/vizicist/palette/engine"
)

func RegisterPlugin(name string, pluginFunc engine.PluginFunc) {
	engine.RegisterPlugin(name, pluginFunc)
}

func init() {
}
