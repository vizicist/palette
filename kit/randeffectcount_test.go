package kit

import (
	"fmt"
	"strings"
	"testing"
)

func TestRandomEffectCountUsesDistinctPlugins(t *testing.T) {
	old := ParamDefs
	t.Cleanup(func() { ParamDefs = old })
	ParamDefs = map[string]ParamDef{}
	for plugin := 0; plugin < 8; plugin++ {
		for bank := 1; bank <= 2; bank++ {
			name := fmt.Sprintf("effect.%d-plugin%d", bank, plugin)
			ParamDefs[name] = ParamDef{Category: "effect", TypedParamDef: ParamDefBool{hasRand: true, randmax: 0.5}}
		}
	}
	ParamDefs["effect.1-optout"] = ParamDef{Category: "effect", TypedParamDef: ParamDefBool{}}
	for _, count := range []int{0, 1, 6, 64} {
		values := map[string]string{"effect.1-plugin0:amount": "0.42"}
		for name := range ParamDefs {
			if name != "effect.1-optout" {
				values[name] = "true"
			}
		}
		setRandomEffectCount(values, count)
		plugins := map[string]bool{}
		enabled := 0
		for key, value := range values {
			if value == "true" && !strings.Contains(key, ":") {
				enabled++
				_, plugin, _ := strings.Cut(key, "-")
				plugins[plugin] = true
			}
		}
		want := count
		if want > 8 {
			want = 8
		}
		if enabled != want || len(plugins) != want {
			t.Errorf("count %d: enabled %d instances of %d types; want %d distinct plugins", count, enabled, len(plugins), want)
		}
		if values["effect.1-optout"] != "false" || values["effect.1-plugin0:amount"] != "0.42" {
			t.Errorf("effect count changed unrelated controls or enabled an opted-out plugin: %+v", values)
		}
	}
}
