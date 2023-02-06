package engine

type PluginFunc func(ctx *PluginContext, api string, apiargs map[string]string) (string, error)

type PluginContext struct {
	api PluginFunc
	// params        *ParamValues
	sources map[string]bool
}

func (ctx *PluginContext) AllowSource(source ...string) {
	var ok bool
	for _, name := range source {
		_, ok = ctx.sources[name]
		if ok {
			LogInfo("AllowSource: already set?", "source", name)
		} else {
			ctx.sources[name] = true
		}
	}
}

func (ctx *PluginContext) IsSourceAllowed(source string) bool {
	_, ok := ctx.sources[source]
	return ok
}