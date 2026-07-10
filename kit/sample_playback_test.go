package kit

import (
	"os"
	"path/filepath"
	"testing"

	ss "github.com/vizicist/palette/pkg/samplesplitter"
)

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

func TestSamplePlaybackPitchBendUsesCursorY(t *testing.T) {
	tests := []struct {
		name string
		y    float64
		want int
	}{
		{name: "bottom", y: 0, want: 0},
		{name: "middle", y: 0.5, want: MidiPitchBendCenter},
		{name: "top", y: 1, want: 16383},
		{name: "below clamps", y: -1, want: 0},
		{name: "above clamps", y: 2, want: 16383},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := samplePlaybackPitchBendFromCursor(CursorEvent{Pos: CursorPos{Y: tt.y}})
			if got != tt.want {
				t.Fatalf("pitch bend for Y=%v = %d, want %d", tt.y, got, tt.want)
			}
		})
	}
}

func TestSamplePlaybackReverbLengthDefaultsWhenParamsUnavailable(t *testing.T) {
	oldGlobalParams := GlobalParams
	defer func() { GlobalParams = oldGlobalParams }()

	GlobalParams = nil
	got := samplePlaybackReverbLength()
	if got != ss.DefaultReverbLength {
		t.Fatalf("reverb length = %v, want %v", got, ss.DefaultReverbLength)
	}
}

func TestClampSamplePlaybackWordThreshold(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "below", in: -0.1, want: 0},
		{name: "inside", in: 0.25, want: 0.25},
		{name: "above", in: 2, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampSamplePlaybackWordThreshold(tt.in)
			if got != tt.want {
				t.Fatalf("clampSamplePlaybackWordThreshold(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSamplesplitterMP3DirUsesConfigBSSDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "data_default", "config", "mp3", "bss")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PALETTE_DATAROOT", root)
	t.Setenv("PALETTE_DATA", "default")

	got := SamplesplitterMP3Dir("")
	if got != dir {
		t.Fatalf("SamplesplitterMP3Dir = %q, want %q", got, dir)
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
