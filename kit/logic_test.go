package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPitchSetIndexForXScalesAcrossPitchCount(t *testing.T) {
	tests := []struct {
		name       string
		x          float64
		pitchCount int
		want       int
	}{
		{name: "left edge", x: 0.0, pitchCount: 42, want: 0},
		{name: "first eighth of forty two", x: 1.0 / 8.0, pitchCount: 42, want: 5},
		{name: "middle of forty two", x: 0.5, pitchCount: 42, want: 21},
		{name: "right edge", x: 1.0, pitchCount: 42, want: 41},
		{name: "stylusrmx still spans eight slots", x: 7.0 / 8.0, pitchCount: 8, want: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pitchSetIndexForX(tt.x, tt.pitchCount)
			if got != tt.want {
				t.Fatalf("pitchSetIndexForX(%v, %d) = %d, want %d", tt.x, tt.pitchCount, got, tt.want)
			}
		})
	}
}

func TestPitchSetIndexForXUsesPreferences(t *testing.T) {
	preferences := []float64{1.0, 2.0, 1.0}
	tests := []struct {
		name string
		x    float64
		want int
	}{
		{name: "first pitch", x: 0.24, want: 0},
		{name: "second pitch starts after one quarter", x: 0.25, want: 1},
		{name: "second pitch spans two quarters", x: 0.74, want: 1},
		{name: "third pitch starts after three quarters", x: 0.75, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pitchSetIndexForX(tt.x, len(preferences), preferences)
			if got != tt.want {
				t.Fatalf("pitchSetIndexForX(%v, %d, %v) = %d, want %d", tt.x, len(preferences), preferences, got, tt.want)
			}
		})
	}
}

func TestLoadPitchSetsConfigDefaultsPreference(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "data_default", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PALETTE_DATAROOT", root)
	t.Setenv("PALETTE_DATA", "default")

	config := `{
  "pitchsets": [
    {
      "name": "weighted",
      "pitches": [
        { "pitch": 60 },
        { "pitch": 62, "preference": 2.5 }
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(configDir, "PitchSets.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	pitchSets, _, preferences, err := LoadPitchSetsConfig()
	if err != nil {
		t.Fatalf("LoadPitchSetsConfig err = %v", err)
	}
	if got := pitchSets["weighted"]; len(got) != 2 || got[0] != 60 || got[1] != 62 {
		t.Fatalf("pitchSets[weighted] = %v, want [60 62]", got)
	}
	got := preferences["weighted"]
	if len(got) != 2 || got[0] != 1.0 || got[1] != 2.5 {
		t.Fatalf("preferences[weighted] = %v, want [1 2.5]", got)
	}
}
