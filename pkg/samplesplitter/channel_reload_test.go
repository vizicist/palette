package samplesplitter

import "testing"

func loadedRotationState() *State {
	s := NewState(Config{BaseNote: 48})
	cue := CueData{Splits: []float64{0}, PeakStarts: []float64{0}, Duration: 1}
	s.SetChannelPlayback(0, WholeSplitMode, false)
	s.SetChannelRotate(0, true)
	s.SetChannelDir(0, `C:\mp3\goat`)
	s.ChannelRotation[0] = []SampleState{{CurrentFile: "goat1.mp3", CueData: &cue}}
	return s
}

func TestChannelLoadedWithSkipsIdenticalRequest(t *testing.T) {
	s := loadedRotationState()
	if !s.ChannelLoadedWith(0, `C:\mp3\goat`, WholeSplitMode, false, true) {
		t.Fatal("an identical reload request was not recognized as a no-op")
	}
}

func TestChannelLoadedWithDetectsChanges(t *testing.T) {
	cases := []struct {
		name   string
		dir    string
		mode   string
		loop   bool
		rotate bool
	}{
		{"different dir", `C:\mp3\bss`, WholeSplitMode, false, true},
		{"different mode", `C:\mp3\goat`, DefaultSplitMode, false, true},
		{"different loop", `C:\mp3\goat`, WholeSplitMode, true, true},
		{"rotation off", `C:\mp3\goat`, WholeSplitMode, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := loadedRotationState()
			if s.ChannelLoadedWith(0, tc.dir, tc.mode, tc.loop, tc.rotate) {
				t.Fatalf("%s was treated as unchanged", tc.name)
			}
		})
	}
}

// Matching settings are not enough - if nothing is loaded we must still load.
func TestChannelLoadedWithRequiresLoadedSamples(t *testing.T) {
	s := NewState(Config{BaseNote: 48})
	s.SetChannelPlayback(0, WholeSplitMode, false)
	s.SetChannelRotate(0, true)
	s.SetChannelDir(0, `C:\mp3\goat`)
	// Settings recorded, but the rotation set is empty.
	if s.ChannelLoadedWith(0, `C:\mp3\goat`, WholeSplitMode, false, true) {
		t.Fatal("an empty rotation set was treated as already loaded")
	}
}

func TestChannelLoadedWithNonRotatingChannel(t *testing.T) {
	s := NewState(Config{BaseNote: 48})
	cue := CueData{Splits: []float64{0}, Duration: 1}
	s.SetChannelPlayback(0, DefaultSplitMode, true)
	s.SetChannelRotate(0, false)
	s.SetChannelDir(0, `C:\mp3\bss`)
	s.ChannelSamples[0] = SampleState{CurrentFile: "x.mp3", CueData: &cue}

	if !s.ChannelLoadedWith(0, `C:\mp3\bss`, DefaultSplitMode, true, false) {
		t.Fatal("an identical non-rotating reload was not recognized as a no-op")
	}
}

func TestChannelLoadedWithFalseForUntouchedChannel(t *testing.T) {
	s := NewState(Config{BaseNote: 48})
	if s.ChannelLoadedWith(3, `C:\mp3\goat`, WholeSplitMode, false, true) {
		t.Fatal("a channel that was never loaded reported as loaded")
	}
}

func TestClearChannelSampleForcesReload(t *testing.T) {
	s := loadedRotationState()
	s.ClearChannelSample(0)
	if s.ChannelLoadedWith(0, `C:\mp3\goat`, WholeSplitMode, false, true) {
		t.Fatal("a cleared channel still reported as loaded")
	}
}
