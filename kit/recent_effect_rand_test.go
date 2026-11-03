package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRecentEffectsParticipateInRandomPatches(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "data_default", "config", "paramdefs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]json.RawMessage
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	for name := range source {
		if !strings.HasPrefix(name, "effect.") {
			delete(source, name)
		}
	}
	data, err = json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	config := filepath.Join(root, "data_effectrand", "config")
	if err := os.MkdirAll(config, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "paramdefs.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	oldDefs, oldGlobal := ParamDefs, GlobalParams
	t.Cleanup(func() { ParamDefs, GlobalParams = oldDefs, oldGlobal })
	t.Setenv("PALETTE_DATAROOT", root)
	t.Setenv("PALETTE_DATA", "effectrand")
	GlobalParams = nil // Exercise the default six distinct plugins.
	if err := LoadParamDefs(); err != nil {
		t.Fatal(err)
	}

	recent := []string{
		"automask", "bendoscope", "bloom", "colorize", "colorpass", "cubetiles",
		"delayrgb", "dilate", "fisheye", "heat", "invertrgb", "kuwahara",
		"lorezeffect", "particlesystem", "pointgrid", "recolour", "shiftglitch",
		"shiftrgb", "stripper", "tunnel",
	}
	for _, plugin := range recent {
		candidates := map[string]string{}
		for bank := 1; bank <= 2; bank++ {
			name := fmt.Sprintf("effect.%d-%s", bank, plugin)
			def, ok := ParamDefs[name]
			if !ok || def.Init != "false" {
				t.Fatalf("%s must exist and start bypassed", name)
			}
			value := RandomValueForParam(def)
			if value == "" {
				t.Fatalf("%s is opted out of Random", name)
			}
			candidates[name] = value
			for parameter, control := range ParamDefs {
				if !strings.HasPrefix(parameter, name+":") {
					continue
				}
				value := RandomValueForParam(control)
				if value == "" {
					t.Fatalf("%s has no Random configuration", parameter)
				}
				switch typed := control.TypedParamDef.(type) {
				case ParamDefFloat:
					n, err := strconv.ParseFloat(value, 64)
					if err != nil || n < typed.min-1e-6 || n > typed.max+1e-6 || typed.randmin < typed.min || typed.randmax > typed.max {
						t.Fatalf("%s generated an invalid value or range: %s, %+v", parameter, value, typed)
					}
				case ParamDefInt:
					n, err := strconv.Atoi(value)
					if err != nil || n < typed.min || n > typed.max || typed.randmin < typed.min || typed.randmax > typed.max {
						t.Fatalf("%s generated an invalid choice or range: %s, %+v", parameter, value, typed)
					}
				}
			}
		}
		// Restrict the candidates to this plugin to prove its selection is
		// possible without depending on a lucky draw from the full pool.
		setRandomEffectCount(candidates, 1)
		enabled := 0
		for name, value := range candidates {
			if value == "true" {
				enabled++
				if !strings.HasSuffix(name, "-"+plugin) {
					t.Fatalf("selected %s instead of %s", name, plugin)
				}
			}
		}
		if enabled != 1 {
			t.Fatalf("%s must be selectable in exactly one bank; got %d", plugin, enabled)
		}
	}
	for draw := 0; draw < 64; draw++ {
		values := randomParamsOnce("effect")
		selected := map[string]bool{}
		for name, value := range values {
			if value != "true" || strings.Contains(name, ":") {
				continue
			}
			_, plugin, _ := strings.Cut(name, "-")
			if selected[plugin] {
				t.Fatalf("Random enabled both banks of %s", plugin)
			}
			selected[plugin] = true
		}
		if len(selected) != 6 {
			t.Fatalf("Random selected %d distinct plugins; want 6", len(selected))
		}
		if values["effect.1-particlesystem:spawnmode"] != "0" {
			t.Fatal("Random must keep particle emission automatic")
		}
	}
}
