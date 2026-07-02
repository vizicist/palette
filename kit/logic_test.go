package kit

import (
	"os"
	"path/filepath"
	"sync/atomic"
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

func setupRetriggerLogicTest(t *testing.T) (*PatchLogic, *ActiveCursor, func()) {
	t.Helper()

	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	oldPatchs := Patchs
	oldSynths := Synths
	oldCursorManager := theCursorManager
	oldScheduler := theScheduler
	oldEngine := theEngine
	oldRouter := theRouter
	oldStepper := theStepper
	oldLog := theLog
	oldTempoFactor := TempoFactor

	cleanup := func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
		Patchs = oldPatchs
		Synths = oldSynths
		theCursorManager = oldCursorManager
		theScheduler = oldScheduler
		theEngine = oldEngine
		theRouter = oldRouter
		theStepper = oldStepper
		theLog = oldLog
		TempoFactor = oldTempoFactor
	}

	InitLog("")
	InitializeClicks()
	SetCurrentClick(100)
	TempoFactor = 1.0

	ParamDefs = map[string]ParamDef{
		"global.mode": {
			Category:      "global",
			Init:          "pro",
			TypedParamDef: ParamDefString{},
		},
		"global.scale": {
			Category:      "global",
			Init:          "",
			TypedParamDef: ParamDefString{},
		},
		"global.deltaztrignote": {
			Category:      "global",
			Init:          "0.2",
			TypedParamDef: ParamDefFloat{min: 0, max: 1},
		},
		"global.deltaztrigcontroller": {
			Category:      "global",
			Init:          "0.02",
			TypedParamDef: ParamDefFloat{min: 0, max: 1},
		},
		"global.deltaytrig": {
			Category:      "global",
			Init:          "0.08",
			TypedParamDef: ParamDefFloat{min: 0, max: 1},
		},
		"misc.quantstyle": {
			Category:      "misc",
			Init:          "none",
			TypedParamDef: ParamDefString{},
		},
		"misc.scale": {
			Category:      "misc",
			Init:          "chromatic",
			TypedParamDef: ParamDefString{},
		},
		"misc.volstyle": {
			Category:      "misc",
			Init:          "pressure",
			TypedParamDef: ParamDefString{},
		},
		"sound.pitchset": {
			Category:      "sound",
			Init:          "",
			TypedParamDef: ParamDefString{},
		},
		"sound.pitchmin": {
			Category:      "sound",
			Init:          "60",
			TypedParamDef: ParamDefInt{min: 0, max: 127},
		},
		"sound.pitchmax": {
			Category:      "sound",
			Init:          "60",
			TypedParamDef: ParamDefInt{min: 0, max: 127},
		},
		"sound.velocitymin": {
			Category:      "sound",
			Init:          "100",
			TypedParamDef: ParamDefInt{min: 0, max: 127},
		},
		"sound.velocitymax": {
			Category:      "sound",
			Init:          "100",
			TypedParamDef: ParamDefInt{min: 0, max: 127},
		},
		"sound.controllerstyle": {
			Category:      "sound",
			Init:          "",
			TypedParamDef: ParamDefString{},
		},
		"sound.samplesplitter": {
			Category:      "sound",
			Init:          "false",
			TypedParamDef: ParamDefBool{},
		},
		"stepper.route": {
			Category:      "stepper",
			Init:          "bidule",
			TypedParamDef: ParamDefString{},
		},
	}

	GlobalParams = NewParamValues()
	for name, def := range ParamDefs {
		if def.Category == "global" {
			if err := GlobalParams.SetParamWithString(name, def.Init); err != nil {
				cleanup()
				t.Fatalf("set global param %s: %v", name, err)
			}
		}
	}

	Patchs = map[string]*Patch{}
	Synths = map[string]*Synth{"": {name: ""}}
	theCursorManager = NewCursorManager()
	theScheduler = NewScheduler()
	theEngine = &Engine{currentPitchOffset: &atomic.Int32{}}
	theRouter = NewRouter()
	theStepper = nil

	patch := NewPatch("A")
	logic := NewPatchLogic(patch)
	ce := NewCursorEvent(7, "A", "down", CursorPos{X: 0.5, Y: 0.20, Z: 0.5})
	ac := &ActiveCursor{Current: ce, Previous: ce, Patch: patch}
	theCursorManager.activeCursors[ce.GID] = ac

	return logic, ac, cleanup
}

func TestRetriggerUsesAccumulatedYSinceLastNoteOn(t *testing.T) {
	logic, ac, cleanup := setupRetriggerLogicTest(t)
	defer cleanup()

	logic.generateSoundFromCursorRetrigger(ac.Current)
	if got := pendingCountOf[*NoteOn](theScheduler); got != 1 {
		t.Fatalf("pending NoteOn after down = %d, want 1", got)
	}

	firstDrag := NewCursorEvent(ac.Current.GID, "A", "drag", CursorPos{X: 0.5, Y: 0.24, Z: 0.5})
	ac.Previous = ac.Current
	ac.Current = firstDrag
	logic.generateSoundFromCursorRetrigger(firstDrag)
	if got := pendingCountOf[*NoteOn](theScheduler); got != 1 {
		t.Fatalf("pending NoteOn after small drag = %d, want 1", got)
	}

	secondDrag := NewCursorEvent(ac.Current.GID, "A", "drag", CursorPos{X: 0.5, Y: 0.29, Z: 0.5})
	ac.Previous = ac.Current
	ac.Current = secondDrag
	logic.generateSoundFromCursorRetrigger(secondDrag)
	if got := pendingCountOf[*NoteOn](theScheduler); got != 2 {
		t.Fatalf("pending NoteOn after accumulated Y drag = %d, want 2", got)
	}
	if got := pendingCountOf[*NoteOff](theScheduler); got != 1 {
		t.Fatalf("pending NoteOff after accumulated Y drag = %d, want 1", got)
	}
	if ac.NoteOnPos != secondDrag.Pos {
		t.Fatalf("NoteOnPos = %+v, want %+v", ac.NoteOnPos, secondDrag.Pos)
	}
}

func pendingCountOf[T any](sched *Scheduler) int {
	count := 0
	sched.pendingMutex.RLock()
	defer sched.pendingMutex.RUnlock()
	for _, se := range sched.pendingScheduled {
		if _, ok := se.Value.(T); ok {
			count++
		}
	}
	return count
}
