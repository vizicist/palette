package samplesplitter

import "testing"

func TestDetectSplitsFixed(t *testing.T) {
	got := detectSplitsFixed(3.1, 1.0)
	want := []float64{0, 1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComputePeakStarts(t *testing.T) {
	samples := []float64{
		0.1, -0.2, 0.8, 0.1,
		0.1, -0.9, 0.2, 0.1,
	}
	got := computePeakStarts(samples, 4, []float64{0, 1}, 2)
	want := []float64{0.5, 1.25}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestGroupWordSplits(t *testing.T) {
	got := groupWordSplits([]float64{0, 0.5, 1.0, 1.5, 2.0}, 2)
	want := []float64{0, 1.0, 2.0}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComputeFirstWordPeakStarts(t *testing.T) {
	samples := []float64{
		0.1, 0.9, 0.1, 0.1,
		0.1, 0.1, 0.95, 0.1,
	}
	got := computeFirstWordPeakStarts(samples, 4, []float64{0}, []float64{0, 1}, 2)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != 0.25 {
		t.Fatalf("got[0] = %v, want first-word peak 0.25", got[0])
	}
}

func TestDetectSplitsWordsStartsAtZero(t *testing.T) {
	frameRate := 1000
	samples := make([]float64, frameRate*2)
	for i := 200; i < 600; i++ {
		samples[i] = 0.5
	}
	for i := 900; i < 1300; i++ {
		samples[i] = 0.5
	}

	got := detectSplitsWords(samples, frameRate, 2, 0.01, 0.12, 0.16, 0.65)
	if len(got) == 0 || got[0] != 0 {
		t.Fatalf("word splits should start at zero, got %v", got)
	}
}

func TestDetectSplitsWordsDoesNotSplitModerateOneShotDip(t *testing.T) {
	frameRate := 1000
	samples := make([]float64, frameRate*2)
	for i := 100; i < 450; i++ {
		samples[i] = 0.7
	}
	for i := 450; i < 750; i++ {
		samples[i] = 0.42
	}
	for i := 750; i < 1100; i++ {
		samples[i] = 0.55
	}
	for i := 1100; i < 1360; i++ {
		samples[i] = 0.3
	}

	got := detectSplitsWords(samples, frameRate, 2, 0.01, 0.12, 0.16, 0.65)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("word splits = %v, want a single split at 0", got)
	}
}
