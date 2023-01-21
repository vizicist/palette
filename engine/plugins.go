package engine

import (
	"fmt"
	"sync"
)

var Plugins map[string]*PluginContext
var PluginsMutex sync.Mutex

func init() {
	Plugins = make(map[string]*PluginContext)
}

func RegisterPlugin(name string, plugin PluginFunc) {
	PluginsMutex.Lock()
	defer PluginsMutex.Unlock()
	_, ok := Plugins[name]
	if ok {
		LogWarn("RegisterPlugin: existing plugin", "plugin", name)
	} else {
		Plugins[name] = NewPluginContext(plugin)
	}
}

func NewPluginContext(apiFunc PluginFunc) *PluginContext {
	return &PluginContext{
		api:     apiFunc,
		sources: map[string]bool{},
	}
}

func PluginsStartPlugin(name string) error {
	PluginsMutex.Lock()
	defer PluginsMutex.Unlock()
	ctx, ok := Plugins[name]
	if !ok {
		return fmt.Errorf("no plugin named %s", name)
	}
	_, err := ctx.api(ctx, "start", nil)
	return err
}

func CallApiOnAllPlugins(api string, apiargs map[string]string) {
	PluginsMutex.Lock()
	defer PluginsMutex.Unlock()
	for _, plugin := range Plugins {
		_, err := plugin.api(plugin, api, apiargs)
		if err != nil {
			LogError(err)
		}
	}
}

func PluginsHandleCursorEvent(ce CursorEvent) {
	PluginsMutex.Lock()
	defer PluginsMutex.Unlock()
	source := ce.Source()
	for _, ctx := range Plugins {
		if ctx.IsSourceAllowed(source) {
			_, err := ctx.api(ctx, "event", ce.ToMap())
			LogError(err)
		}
	}
}

func PluginsHandleMidiEvent(e MidiEvent) {
	PluginsMutex.Lock()
	defer PluginsMutex.Unlock()
	for _, ctx := range Plugins {
		_, err := ctx.api(ctx, "event", e.ToMap())
		LogError(err)
	}
}

func PluginsHandleClickEvent(e ClickEvent) {
	PluginsMutex.Lock()
	defer PluginsMutex.Unlock()
	for _, ctx := range Plugins {
		_, err := ctx.api(ctx, "event", e.ToMap())
		LogError(err)
	}
}
