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
	tests := map[string]string{
		"global.initialpage": "global.mode",
		"global.mode":        "global.mode",
		"global.log":         "global.log",
	}

	for name, want := range tests {
		if got := canonicalGlobalParamName(name); got != want {
			t.Fatalf("canonicalGlobalParamName(%q) = %q, want %q", name, got, want)
		}
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

func TestSetRouteStopsActiveCursorNoteAndPendingSound(t *testing.T) {
	cleanup := setupModeTest(t, "bss")
	defer cleanup()

	oldCursorManager := theCursorManager
	oldScheduler := theScheduler
	oldQuad := theQuad
	oldLog := theLog
	defer func() {
		theCursorManager = oldCursorManager
		theScheduler = oldScheduler
		theQuad = oldQuad
		theLog = oldLog
	}()

	t.Setenv("PALETTE_DATAROOT", t.TempDir())
	InitLog("")
	InitializeClicks()
	theCursorManager = NewCursorManager()
	theScheduler = NewScheduler()
	theQuad = NewQuad()
	stepper := NewStepper()
	theStepper = stepper

	patch := NewPatch("A")
	synth := &Synth{name: "test"}
	patch.synth = synth
	theQuad.patch["A"] = patch
	if err := patch.SetParam("stepper.route", "bidule"); err != nil {
		t.Fatalf("set initial route: %v", err)
	}

	noteOn := NewNoteOn(synth, 60, 100)
	theCursorManager.activeCursors[42] = &ActiveCursor{
		Current:     NewCursorEvent(42, "A", "drag", CursorPos{X: 0.5, Y: 0.5, Z: 0.5}),
		NoteOn:      noteOn,
		NoteOnClick: CurrentClick() + 8,
	}
	ScheduleAt(CurrentClick()+8, "A", noteOn)
	ScheduleAt(CurrentClick()+9, "A", NewNoteOff(synth, 60, 100))
	theScheduler.insertScheduleElement(NewSchedElement(CurrentClick()+10, "A", NewPitchBend(synth, MidiPitchBendCenter)))

	if _, err := stepper.SetRoute("A", "samplesplitter"); err != nil {
		t.Fatalf("SetRoute: %v", err)
	}

	ac := theCursorManager.activeCursors[42]
	if ac == nil {
		t.Fatal("active cursor should remain present after route change")
	}
	if ac.NoteOn != nil {
		t.Fatal("route change did not clear active cursor NoteOn")
	}
	if hasSoundEventWithTag(theScheduler, "A") {
		t.Fatal("route change left pending sound events for patch A")
	}
	if got := stepper.config.RouteForPatch("A"); got != StepperRouteSamplesplitter {
		t.Fatalf("route = %q, want %q", got, StepperRouteSamplesplitter)
	}
}

func hasSoundEventWithTag(sched *Scheduler, tag string) bool {
	sched.pendingMutex.RLock()
	for _, se := range sched.pendingScheduled {
		if se.Tag == tag && isSoundEvent(se.Value) {
			sched.pendingMutex.RUnlock()
			return true
		}
	}
	sched.pendingMutex.RUnlock()

	sched.mutex.RLock()
	defer sched.mutex.RUnlock()
	for i := sched.schedList.Front(); i != nil; i = i.Next() {
		se := i.Value.(*SchedElement)
		if se.Tag == tag && isSoundEvent(se.Value) {
			return true
		}
	}
	return false
}
