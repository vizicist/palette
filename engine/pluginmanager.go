package engine

import (
	"fmt"
)

// type PluginInfo struct {
// }
func ThePluginManager() *PluginManager {
	return TheEngine().PluginManager
}

func NewPluginContext(apiFunc PluginFunc) *PluginContext {
	return &PluginContext{
		api:           apiFunc,
		cursorManager: NewCursorManager(),
		params:        NewParamValues(),
		sources:       map[string]bool{},
	}
}

type PluginManager struct {
	plugins map[string]*PluginContext
}

func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]*PluginContext),
	}
}

func (tm *PluginManager) RegisterPlugin(name string, apiFunc PluginFunc) {
	_, ok := tm.plugins[name]
	if ok {
		LogWarn("RegisterPlugin: existing plugin", "plugin", name)
	} else {
		tm.plugins[name] = NewPluginContext(apiFunc)
		LogInfo("Registering Plugin", "plugin", name)
	}
}

func (tm *PluginManager) StartPlugin(name string) error {
	plugin, err := tm.GetPluginContext(name)
	if err != nil {
		return err
	}
	_, err = plugin.api(plugin, "start", nil)
	return err
}

func CallApiOnAllPlugins(api string, apiargs map[string]string) {
	pm := ThePluginManager()
	for _, plugin := range pm.plugins {
		plugin.api(plugin, api, apiargs)
	}
}

/*
func (rm *PluginManager) handleCursorEvent(ce CursorEvent) {
	for name, plugin := range rm.plugins {
		DebugLogOfType("plugin", "CallPlugins", "name", name)
		context, ok := rm.pluginsContext[name]
		if !ok {
			Warn("PluginManager.handle: no context", "name", name)
		} else {
			plugin.OnCursorEvent(context, ce)
		}
	}
}

func (rm *PluginManager) handleMidiEvent(me MidiEvent) {
	for name, plugin := range rm.plugins {
		context, ok := rm.pluginsContext[name]
		if !ok {
			Warn("PluginManager.handle: no context", "name", name)
		} else {
			plugin.OnMidiEvent(context, me)
		}
	}
}
*/

/*
func (pm *PluginManager) ApplyToPluginsNamed(pluginName string, f func(plugin Plugin)) {
	for name, ctx := range pm.pluginsContext {
		if pluginName == name {
			f(plugin)
		}
	}
}

func (pm *PluginManager) GetPlugin(pluginName string) (Plugin, error) {
	plugin, ok := pm.plugins[pluginName]
	if !ok {
		return nil, fmt.Errorf("no plugin named %s", pluginName)
	} else {
		return plugin, nil
	}
}
*/

func (pm *PluginManager) GetPluginContext(name string) (*PluginContext, error) {
	ctx, ok := pm.plugins[name]
	if !ok {
		return nil, fmt.Errorf("no plugin named %s", name)
	} else {
		return ctx, nil
	}
}

func (tm *PluginManager) handleCursorEvent(e CursorEvent) {
	for _, ctx := range tm.plugins {
		if ctx.IsSourceAllowed(e.Source) {
			ctx.api(ctx, "event", e.ToMap())
		}
	}
}

func (tm *PluginManager) handleMidiEvent(e MidiEvent) {
	for _, ctx := range tm.plugins {
		ctx.api(ctx, "event", e.ToMap())
	}
}

func (tm *PluginManager) handleClickEvent(e ClickEvent) {
	for _, ctx := range tm.plugins {
		ctx.api(ctx, "event", e.ToMap())
	}
}
