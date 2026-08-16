package kit

import (
	"sort"
	"strings"
	"testing"
)

// applied runs ApplyValuesFromMap for a category and reports which parameters
// it decided to set.
func applied(category string, paramsmap map[string]any) []string {
	got := []string{}
	vals := NewParamValues()
	vals.ApplyValuesFromMap(category, paramsmap, func(name string, value string) error {
		got = append(got, name)
		return nil
	})
	sort.Strings(got)
	return got
}

// A themed quad category has to apply per-patch parameters exactly as the bare
// "quad" category does.
//
// patch.load feeds config/paramoverrides.json through ApplyValuesFromMap using
// whatever category was loaded, and the GUI always loads quad presets under a
// theme - defaultThemeDir is quad_default - so a literal category == "quad"
// test meant the overrides applied at boot and during attract but were silently
// dropped on every load a human performed.
func TestApplyValuesFromMapAppliesPerPatchForThemedQuad(t *testing.T) {

	overrides := map[string]any{
		"visual.brightness": "0.4",
		"sound.volume":      "0.8",
		"global.something":  "ignored", // not per-patch, and not this category
	}
	want := "sound.volume,visual.brightness"

	for _, category := range []string{"quad", "quad_default", "quad_chill", "quad_goat"} {
		got := strings.Join(applied(category, overrides), ",")
		if got != want {
			t.Errorf("category %q applied [%s], want [%s]", category, got, want)
		}
	}
}

// The "patch" category keeps behaving as before, and a category that is neither
// patch nor quad still only matches its own parameters.
func TestApplyValuesFromMapCategoryMatchingUnchanged(t *testing.T) {

	paramsmap := map[string]any{
		"visual.brightness": "0.4",
		"sound.volume":      "0.8",
	}

	if got := strings.Join(applied("patch", paramsmap), ","); got != "sound.volume,visual.brightness" {
		t.Errorf(`category "patch" applied [%s]`, got)
	}
	if got := strings.Join(applied("sound", paramsmap), ","); got != "sound.volume" {
		t.Errorf(`category "sound" applied [%s], want only its own`, got)
	}
	if got := strings.Join(applied("global", paramsmap), ","); got != "" {
		t.Errorf(`category "global" applied [%s], want nothing`, got)
	}
}
