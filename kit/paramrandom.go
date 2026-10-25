package kit

import (
	"math/rand"
	"sort"
	"strings"
)

// Ordinary parameter generation and effect-count selection.
func randomParamsOnce(category string) map[string]string {
	m := map[string]string{}
	for name, def := range ParamDefs {
		if def.Category == category || category == "*" {
			if v := RandomValueForParam(def); v != "" {
				m[name] = v
			}
		}
	}
	if category == "effect" || category == "*" {
		count := 6
		if GlobalParams != nil {
			if configured, err := GetParamInt("global.randeffectcount"); err == nil {
				count = configured
			}
		}
		setRandomEffectCount(m, count)
	}
	return m
}

// Choose distinct plugin types and one bank per type. Rand still respects the
// opt-in parameter ranges; effects without randomization enabled are cleared
// rather than leaking in from the previously loaded patch.
func setRandomEffectCount(values map[string]string, count int) {
	groups := map[string][]string{}
	for name, def := range ParamDefs {
		if def.Category != "effect" || strings.Contains(name, ":") {
			continue
		}
		if _, ok := def.TypedParamDef.(ParamDefBool); !ok {
			continue
		}
		if _, eligible := values[name]; eligible {
			_, plugin, found := strings.Cut(strings.TrimPrefix(name, "effect."), "-")
			if found {
				groups[plugin] = append(groups[plugin], name)
			}
		}
		values[name] = "false"
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	if count < 0 {
		count = 0
	}
	if count > len(names) {
		count = len(names)
	}
	for _, plugin := range names[:count] {
		banks := groups[plugin]
		values[banks[rand.Intn(len(banks))]] = "true"
	}
}
