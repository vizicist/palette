package kit

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestRandomValuePreservesConfiguredRangesAndChoices(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		f := RandomValueForParam(ParamDef{TypedParamDef: ParamDefFloat{min: 0, max: 10, randmin: 2.5, randmax: 3.5, hasRand: true}})
		if value, err := strconv.ParseFloat(f, 64); err != nil || value < 2.5 || value > 3.5 {
			t.Fatalf("float outside configured Rand range: %q", f)
		}
		i := RandomValueForParam(ParamDef{TypedParamDef: ParamDefInt{min: 0, max: 20, randmin: 3, randmax: 7, hasRand: true}})
		if value, err := strconv.Atoi(i); err != nil || value < 3 || value > 7 {
			t.Fatalf("integer outside configured Rand range: %q", i)
		}
		choice := RandomValueForParam(ParamDef{TypedParamDef: ParamDefString{values: []string{"circle", "square"}, hasRand: true}})
		if choice != "circle" && choice != "square" {
			t.Fatalf("invalid enum choice: %q", choice)
		}
	}
}

func TestRandomValuesPreservePinnedAndOptedOutSettings(t *testing.T) {
	for _, tc := range []struct {
		name string
		def  any
		want string
	}{
		{"disabled float", ParamDefFloat{min: 0, max: 1}, ""},
		{"disabled integer", ParamDefInt{min: 0, max: 10}, ""},
		{"disabled boolean", ParamDefBool{}, ""},
		{"disabled string", ParamDefString{values: []string{"left", "right"}}, ""},
		{"pinned string", ParamDefString{values: []string{"cursor", "random"}, randmax: "cursor", hasRand: true}, "cursor"},
		{"empty enum", ParamDefString{hasRand: true}, ""},
		{"always false", ParamDefBool{randmax: 0, hasRand: true}, "false"},
		{"always true", ParamDefBool{randmax: 1, hasRand: true}, "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RandomValueForParam(ParamDef{TypedParamDef: tc.def}); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCategoryRandPreservesOptInAndCategoryBoundaries(t *testing.T) {
	old := ParamDefs
	t.Cleanup(func() { ParamDefs = old })
	ParamDefs = map[string]ParamDef{
		"visual.alpha":     {Category: "visual", TypedParamDef: ParamDefFloat{min: 0, max: 1, randmin: 0.5, randmax: 0.5, hasRand: true}},
		"visual.shape":     {Category: "visual", TypedParamDef: ParamDefString{values: []string{"circle"}, hasRand: true}},
		"visual.placement": {Category: "visual", TypedParamDef: ParamDefString{values: []string{"cursor", "random"}, randmax: "cursor", hasRand: true}},
		"visual.unchanged": {Category: "visual", TypedParamDef: ParamDefInt{min: 0, max: 10}},
		"sound.level":      {Category: "sound", TypedParamDef: ParamDefFloat{min: 0, max: 1, randmin: 1, randmax: 1, hasRand: true}},
	}
	raw, err := ParamRandomValuesForCategory("visual")
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 3 || values["visual.alpha"] != "0.500000" || values["visual.shape"] != "circle" || values["visual.placement"] != "cursor" {
		t.Errorf("unexpected category randomization: %v", values)
	}
}
