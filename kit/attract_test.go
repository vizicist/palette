package kit

import (
	"math/rand"
	"testing"
)

func setupAttractParamTest(t *testing.T) func() {
	t.Helper()

	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	oldAttractManager := theAttractManager

	ParamDefs = map[string]ParamDef{
		"global.attractgesturezmin": {
			Category: "global",
			Init:     "0.25",
			TypedParamDef: ParamDefFloat{
				min: 0,
				max: 1,
			},
		},
		"global.attractgesturezmax": {
			Category: "global",
			Init:     "0.75",
			TypedParamDef: ParamDefFloat{
				min: 0,
				max: 1,
			},
		},
	}
	GlobalParams = NewParamValues()
	theAttractManager = &AttractManager{}

	return func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
		theAttractManager = oldAttractManager
	}
}

func TestAttractGestureZParamsApplyLive(t *testing.T) {
	cleanup := setupAttractParamTest(t)
	defer cleanup()

	if err := SetAndApplyGlobalParam("global.attractgesturezmin", "0.40"); err != nil {
		t.Fatalf("SetAndApplyGlobalParam zmin err = %v", err)
	}
	if err := SetAndApplyGlobalParam("global.attractgesturezmax", "0.90"); err != nil {
		t.Fatalf("SetAndApplyGlobalParam zmax err = %v", err)
	}

	if got := theAttractManager.GestureZMin; got != 0.40 {
		t.Fatalf("GestureZMin = %v, want 0.40", got)
	}
	if got := theAttractManager.GestureZMax; got != 0.90 {
		t.Fatalf("GestureZMax = %v, want 0.90", got)
	}
}

func TestAttractGestureZParamsRejectTypoNames(t *testing.T) {
	cleanup := setupAttractParamTest(t)
	defer cleanup()

	if err := SetAndApplyGlobalParam("global.attractgestureminz", "0.40"); err == nil {
		t.Fatal("typo global.attractgestureminz should be rejected")
	}
	if err := SetAndApplyGlobalParam("global.attractgesturemaxz", "0.90"); err == nil {
		t.Fatal("typo global.attractgesturemaxz should be rejected")
	}
}

func TestZRandHandlesInvalidRanges(t *testing.T) {
	cm := &CursorManager{
		cursorRand: rand.New(rand.NewSource(1)),
	}

	if got := cm.zRand(0.45, 0.45); got != 0.45 {
		t.Fatalf("zRand equal range = %v, want 0.45", got)
	}
	if got := cm.zRand(0.8, 0.2); got < 0.2 || got > 0.8 {
		t.Fatalf("zRand inverted range = %v, want within [0.2, 0.8]", got)
	}
	if got := cm.zRand(0, 0); got != 0 {
		t.Fatalf("zRand zero range = %v, want 0", got)
	}
}

func TestNormalizeAttractRangeHandlesInvalidRanges(t *testing.T) {
	minLength, maxLength := normalizeAttractRange(0.8, 0.2)
	if minLength != 0.2 || maxLength != 0.8 {
		t.Fatalf("normalizeAttractRange inverted = (%v, %v), want (0.2, 0.8)", minLength, maxLength)
	}

	minLength, maxLength = normalizeAttractRange(-1, 2)
	if minLength != 0 || maxLength != 1 {
		t.Fatalf("normalizeAttractRange clamped = (%v, %v), want (0, 1)", minLength, maxLength)
	}
}

func TestGenerateRandomGestureHandlesEqualLengthRange(t *testing.T) {
	oldAttractManager := theAttractManager
	defer func() { theAttractManager = oldAttractManager }()

	theAttractManager = &AttractManager{
		GestureMinLength: 0.5,
		GestureMaxLength: 0.5,
		GestureZMin:      0.5,
		GestureZMax:      0.5,
	}
	cm := &CursorManager{
		cursorRand: rand.New(rand.NewSource(1)),
	}

	pos0, pos1 := cm.randomAttractGestureEndpoints()
	if got := lineLength(pos0, pos1); got > 0.5 {
		t.Fatalf("randomAttractGestureEndpoints length = %v, want <= 0.5", got)
	}
}
