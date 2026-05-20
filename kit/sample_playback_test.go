package kit

import "testing"

func TestSamplePlaybackVelocityUsesSamplePlaybackVolume(t *testing.T) {
	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	defer func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
	}()

	ParamDefs = map[string]ParamDef{
		"global.sampleplaybackvolume": {
			Category: "global",
			Init:     "1.0",
			TypedParamDef: ParamDefFloat{
				min:  0,
				max:  2,
				Init: "1.0",
			},
		},
	}
	GlobalParams = NewParamValues()

	if err := GlobalParams.SetParamWithString("global.sampleplaybackvolume", "0.5"); err != nil {
		t.Fatalf("SetParamWithString err = %v", err)
	}
	got := samplePlaybackVelocityFromPressure(nil, 1.0)
	if got != 64 {
		t.Fatalf("velocity = %d, want 64", got)
	}

	if err := GlobalParams.SetParamWithString("global.sampleplaybackvolume", "2.0"); err != nil {
		t.Fatalf("SetParamWithString err = %v", err)
	}
	got = samplePlaybackVelocityFromPressure(nil, 1.0)
	if got != 127 {
		t.Fatalf("velocity = %d, want capped 127", got)
	}
}

func TestSamplePlaybackVolumeDefaultsWhenParamsUnavailable(t *testing.T) {
	oldGlobalParams := GlobalParams
	defer func() { GlobalParams = oldGlobalParams }()

	GlobalParams = nil
	got := samplePlaybackVelocityFromPressure(nil, 1.0)
	if got != 127 {
		t.Fatalf("velocity = %d, want default 127", got)
	}
}

func TestSamplePlaybackQuantUsesCanonicalParam(t *testing.T) {
	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	defer func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
	}()

	ParamDefs = map[string]ParamDef{
		"global.sampleplaybackquant": {
			Category: "global",
			Init:     "0.0",
			TypedParamDef: ParamDefFloat{
				min: 0,
				max: 1,
			},
		},
	}
	GlobalParams = NewParamValues()

	if err := SetAndApplyGlobalParam("global.sampleplaybackquant", "0.25"); err != nil {
		t.Fatalf("SetAndApplyGlobalParam canonical err = %v", err)
	}
	got, err := GetParamFloat("global.sampleplaybackquant")
	if err != nil {
		t.Fatalf("GetParamFloat err = %v", err)
	}
	if got != 0.25 {
		t.Fatalf("global.sampleplaybackquant = %v, want 0.25", got)
	}
	if err := SetAndApplyGlobalParam("global.transmissionquant", "0.25"); err == nil {
		t.Fatal("legacy global.transmissionquant should be rejected")
	}
}

func TestDeleteActiveCursorStopsActiveSamplePlayback(t *testing.T) {
	oldScheduler := theScheduler
	oldLog := theLog
	defer func() { theScheduler = oldScheduler }()
	defer func() { theLog = oldLog }()

	InitLog("")
	InitializeClicks()
	theScheduler = NewScheduler()
	cm := NewCursorManager()
	cm.activeCursors[7] = &ActiveCursor{
		Current: NewCursorEvent(7, "B", "drag", CursorPos{X: 0.5, Y: 0.5, Z: 0.5}),
		ActiveSamplePlayback: &ActiveSamplePlayback{
			Patch:          "B",
			SigilChannel:   1,
			SampleSelector: 12,
			VoiceKey:       "cursor-B-7",
		},
	}

	cm.DeleteActiveCursor(7)

	if _, ok := cm.activeCursors[7]; ok {
		t.Fatal("active cursor was not deleted")
	}
	var sawStop bool
	theScheduler.pendingMutex.RLock()
	for _, se := range theScheduler.pendingScheduled {
		stop, ok := se.Value.(*SamplePlaybackStop)
		if ok && stop.VoiceKey == "cursor-B-7" {
			sawStop = true
			break
		}
	}
	theScheduler.pendingMutex.RUnlock()
	if !sawStop {
		t.Fatal("DeleteActiveCursor did not schedule SamplePlaybackStop")
	}
}
