package kit

import "testing"

func setupModeTest(t *testing.T, mode string) func() {
	t.Helper()

	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	oldPatchs := Patchs
	oldSynths := Synths
	oldStepper := theStepper
	oldProcessManager := theProcessManager
	oldNoProcess := NoProcess

	ParamDefs = map[string]ParamDef{
		"global.mode": {
			Category:      "global",
			Init:          "pro",
			TypedParamDef: ParamDefString{},
		},
		"stepper.route": {
			Category:      "stepper",
			Init:          "samplesplitter",
			TypedParamDef: ParamDefString{},
		},
		"visual.shape": {
			Category:      "visual",
			Init:          "square",
			TypedParamDef: ParamDefString{},
		},
	}
	GlobalParams = NewParamValues()
	if err := GlobalParams.SetParamWithString("global.mode", mode); err != nil {
		t.Fatalf("set mode: %v", err)
	}
	Patchs = map[string]*Patch{}
	Synths = map[string]*Synth{"": {name: ""}}
	theStepper = nil
	theProcessManager = nil
	NoProcess = true

	return func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
		Patchs = oldPatchs
		Synths = oldSynths
		theStepper = oldStepper
		theProcessManager = oldProcessManager
		NoProcess = oldNoProcess
	}
}

func TestModeControlsPageSelection(t *testing.T) {
	cleanup := setupModeTest(t, "pro")
	defer cleanup()

	if IsBSSInitialPage() {
		t.Fatal("pro mode should not be treated as BSS")
	}
	if got := CurrentMode(); got != "pro" {
		t.Fatalf("CurrentMode = %q, want pro", got)
	}

	if err := SetAndApplyGlobalParam("global.mode", "bss"); err != nil {
		t.Fatalf("SetAndApplyGlobalParam: %v", err)
	}
	if !IsBSSInitialPage() {
		t.Fatal("bss mode should be treated as BSS")
	}
	if got := CurrentMode(); got != "bss" {
		t.Fatalf("CurrentMode = %q, want bss", got)
	}
}

func TestLegacyInitialPageMigratesToMode(t *testing.T) {
	if got := canonicalGlobalParamName("global.initialpage"); got != "global.mode" {
		t.Fatalf("canonicalGlobalParamName = %q, want global.mode", got)
	}
}

func TestNewStepperStartsDisarmed(t *testing.T) {
	stepper := NewStepper()
	if stepper.playing {
		t.Fatal("new stepper should not start playing")
	}
	for patch, track := range stepper.tracks {
		if track.Recording {
			t.Fatalf("new stepper track %s should not be recording", patch)
		}
	}
}

func TestProModeApplyDisarmsStepper(t *testing.T) {
	cleanup := setupModeTest(t, "pro")
	defer cleanup()

	stepper := NewStepper()
	theStepper = stepper
	if _, err := stepper.SetRecording("A", true); err != nil {
		t.Fatalf("SetRecording: %v", err)
	}
	if !stepper.playing {
		t.Fatal("pro recording should start stepper playback")
	}

	Synths = nil
	if err := SetAndApplyGlobalParam("global.mode", "pro"); err != nil {
		t.Fatalf("SetAndApplyGlobalParam: %v", err)
	}
	if stepper.playing {
		t.Fatal("pro mode apply should stop stepper playback")
	}
	for patch, track := range stepper.tracks {
		if track.Recording {
			t.Fatalf("pro mode apply should disarm track %s", patch)
		}
	}
}

func TestBSSRecordingDoesNotStartStepperSequencing(t *testing.T) {
	cleanup := setupModeTest(t, "bss")
	defer cleanup()

	stepper := NewStepper()
	theStepper = stepper
	if _, err := stepper.SetRecording("A", true); err != nil {
		t.Fatalf("SetRecording: %v", err)
	}
	if stepper.playing {
		t.Fatal("bss recording should not start stepper sequencing")
	}
}

func TestProModeCoercesSamplesplitterRoutes(t *testing.T) {
	cleanup := setupModeTest(t, "pro")
	defer cleanup()

	patch := NewPatch("A")
	if err := patch.SetParam("stepper.route", "both"); err != nil {
		t.Fatalf("set stepper.route: %v", err)
	}

	config := NewStepperConfig()
	if got := config.RouteForPatch("A"); got != StepperRouteBidule {
		t.Fatalf("pro RouteForPatch = %q, want bidule", got)
	}
	if config.RouteIncludesSamples("A") {
		t.Fatal("pro mode should not include samples in route")
	}

	if err := SetAndApplyGlobalParam("global.mode", "bss"); err != nil {
		t.Fatalf("SetAndApplyGlobalParam: %v", err)
	}
	if got := config.RouteForPatch("A"); got != StepperRouteBoth {
		t.Fatalf("bss RouteForPatch = %q, want both", got)
	}
	if !config.RouteIncludesSamples("A") {
		t.Fatal("bss mode should allow samples route")
	}
}

func TestProModeSwallowsDirectSamplesplitterSynthSends(t *testing.T) {
	cleanup := setupModeTest(t, "pro")
	defer cleanup()

	synth := &Synth{
		name:        "sample",
		portchannel: PortChannel{port: samplesplitterMidiPort, channel: 1},
	}

	if !SendSamplesplitterNote(synth, NewNoteOn(synth, 60, 100)) {
		t.Fatal("pro mode should swallow SampleSplitter notes")
	}
	if !SendSamplesplitterPitchBend(synth, MidiPitchBendCenter) {
		t.Fatal("pro mode should swallow SampleSplitter pitch bends")
	}
}
