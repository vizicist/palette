package kit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func loadRandChoicesTestDef(t *testing.T, extra map[string]any) error {
	t.Helper()
	oldDefs, oldEnums := ParamDefs, ParamEnums
	t.Cleanup(func() { ParamDefs, ParamEnums = oldDefs, oldEnums })
	ParamEnums = map[string][]string{"shape": {"line", "circle", "square", "triangle", "star"}}
	root := t.TempDir()
	t.Setenv("PALETTE_DATAROOT", root)
	t.Setenv("PALETTE_DATA", "randchoices")
	dir := filepath.Join(root, "data_randchoices", "config")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	spec := map[string]any{"valuetype": "string", "min": "shape", "max": "shape", "init": "square"}
	for key, value := range extra {
		spec[key] = value
	}
	data, err := json.Marshal(map[string]any{"visual.shape": spec})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "paramdefs.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return LoadParamDefs()
}

func TestRandChoicesLimitsRandAndPreservesOtherShapes(t *testing.T) {
	choices := []string{"line", "circle", "square", "triangle"}
	if err := loadRandChoicesTestDef(t, map[string]any{"randchoices": choices}); err != nil {
		t.Fatal(err)
	}
	def := ParamDefs["visual.shape"]
	typed := def.TypedParamDef.(ParamDefString)
	if !reflect.DeepEqual(typed.values, ParamEnums["shape"]) || !reflect.DeepEqual(typed.randchoices, choices) {
		t.Fatalf("random subset replaced the full shape list: %+v", typed)
	}
	seen := map[string]bool{}
	for i := 0; i < 256; i++ {
		value := RandomValueForParam(def)
		if value != "line" && value != "circle" && value != "square" && value != "triangle" {
			t.Fatalf("Rand selected a shape outside the subset: %q", value)
		}
		seen[value] = true
	}
	if len(seen) != len(choices) {
		t.Errorf("not all configured choices were reachable: %v", seen)
	}
	values := NewParamValues()
	if err := values.SetParamWithString("visual.shape", "star"); err != nil {
		t.Fatalf("nonrandom selection outside the subset failed: %v", err)
	}
	if value, err := values.Get("visual.shape"); err != nil || value != "star" {
		t.Errorf("nonrandom selection outside the subset changed: %q, %v", value, err)
	}
}

func TestRandChoicesPreservesPinnedValue(t *testing.T) {
	if err := loadRandChoicesTestDef(t, map[string]any{"randchoices": []string{"line"}, "randmax": "star"}); err != nil {
		t.Fatal(err)
	}
	if got := RandomValueForParam(ParamDefs["visual.shape"]); got != "star" {
		t.Errorf("nonempty randmax must still pin its value: %q", got)
	}
}

func TestRandChoicesOmittedPreservesFullEnum(t *testing.T) {
	if err := loadRandChoicesTestDef(t, map[string]any{"randmax": ""}); err != nil {
		t.Fatal(err)
	}
	typed := ParamDefs["visual.shape"].TypedParamDef.(ParamDefString)
	if typed.randchoices != nil || !reflect.DeepEqual(typed.values, ParamEnums["shape"]) {
		t.Errorf("omitted subset changed the full list: %+v", typed)
	}
	seenStar := false
	for i := 0; i < 256; i++ {
		seenStar = seenStar || RandomValueForParam(ParamDefs["visual.shape"]) == "star"
	}
	if !seenStar {
		t.Error("full-enum Rand never selected a shape outside the usual subset")
	}
}

func TestRandChoicesRejectsInvalidDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		choices any
	}{
		{"null", nil},
		{"not an array", "line"},
		{"empty", []string{}},
		{"non-string", []any{"line", 1}},
		{"unknown shape", []string{"missing"}},
		{"duplicate", []string{"line", "line"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := loadRandChoicesTestDef(t, map[string]any{"randchoices": tc.choices})
			if err == nil || !strings.Contains(err.Error(), "visual.shape") || !strings.Contains(err.Error(), "randchoices") {
				t.Errorf("expected a clear parameter error, got %v", err)
			}
		})
	}
}
