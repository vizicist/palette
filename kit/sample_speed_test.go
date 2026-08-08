package kit

import (
	"math"
	"testing"
)

// bendToSpeed mirrors what the service does with the bend value: +/-12
// semitones across the 14-bit range, then 2^(semitones/12) as the playback
// rate. Expressing assertions in speed rather than bend units keeps them
// readable.
func bendToSpeed(bend int) float64 {
	semitones := (float64(bend-8192) / 8192.0) * 12.0
	return math.Pow(2, semitones/12.0)
}

func closeTo(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// speedOffsetToBend mirrors the scaling in samplePlaybackPitchBendFromCursor.
func speedOffsetToBend(offset float64) int {
	scale := float64(16383 - MidiPitchBendCenter)
	if offset < 0 {
		scale = float64(MidiPitchBendCenter)
	}
	return int(math.Round(float64(MidiPitchBendCenter) + offset*scale))
}

func TestSpeedOffsetSpansFullRangeFromCenter(t *testing.T) {
	// With the center at the middle, the two halves are symmetric.
	if got := samplePlaybackSpeedOffset(0.0, 0.5); !closeTo(got, -1, 1e-9) {
		t.Fatalf("bottom offset = %v, want -1", got)
	}
	if got := samplePlaybackSpeedOffset(0.5, 0.5); !closeTo(got, 0, 1e-9) {
		t.Fatalf("center offset = %v, want 0", got)
	}
	if got := samplePlaybackSpeedOffset(1.0, 0.5); !closeTo(got, 1, 1e-9) {
		t.Fatalf("top offset = %v, want 1", got)
	}
}

// Raising the center gives more of the pad to slower speeds without losing
// the fast end - that is the point of the parameter.
func TestSpeedCenterReallocatesThePad(t *testing.T) {
	const center = 0.8

	if got := samplePlaybackSpeedOffset(center, center); !closeTo(got, 0, 1e-9) {
		t.Fatalf("offset at the center = %v, want 0", got)
	}
	if got := samplePlaybackSpeedOffset(1.0, center); !closeTo(got, 1, 1e-9) {
		t.Fatalf("top offset = %v, want 1 (the fast end is still reachable)", got)
	}
	if got := samplePlaybackSpeedOffset(0.0, center); !closeTo(got, -1, 1e-9) {
		t.Fatalf("bottom offset = %v, want -1", got)
	}
	// The midpoint of the pad is now below normal speed.
	if got := samplePlaybackSpeedOffset(0.5, center); got >= 0 {
		t.Fatalf("mid-pad offset = %v, want negative with a high center", got)
	}
}

func TestSpeedOffsetHandlesDegenerateCenters(t *testing.T) {
	if got := samplePlaybackSpeedOffset(0.5, 0); !closeTo(got, 0.5, 1e-9) {
		t.Fatalf("center 0: offset = %v, want 0.5", got)
	}
	if got := samplePlaybackSpeedOffset(0.5, 1); !closeTo(got, -0.5, 1e-9) {
		t.Fatalf("center 1: offset = %v, want -0.5", got)
	}
}

func TestQuantizeSpeedKeepsBothExtremes(t *testing.T) {
	// 5 divisions across the axis => 0, .25, .5, .75, 1
	want := []float64{0, 0.25, 0.5, 0.75, 1}
	got := make([]float64, 0, len(want))
	for _, y := range []float64{0.0, 0.3, 0.5, 0.7, 1.0} {
		got = append(got, quantizeSamplePlaybackSpeed(y, 5))
	}
	for i := range want {
		if !closeTo(got[i], want[i], 1e-9) {
			t.Fatalf("quantized = %v, want %v", got, want)
		}
	}
}

func TestQuantizeSpeedOddDivisionsIncludeTheCenter(t *testing.T) {
	// An odd division count must be able to land exactly on normal speed.
	if got := quantizeSamplePlaybackSpeed(0.5, 5); !closeTo(got, 0.5, 1e-9) {
		t.Fatalf("mid-pad with 5 divisions = %v, want exactly 0.5", got)
	}
}

func TestQuantizeSpeedOffBelowTwoDivisions(t *testing.T) {
	for _, n := range []int{0, 1, -3} {
		if got := quantizeSamplePlaybackSpeed(0.37, n); !closeTo(got, 0.37, 1e-9) {
			t.Fatalf("divisions=%d changed the value to %v, want it untouched", n, got)
		}
	}
}

func TestQuantizeSpeedNeverExceedsRange(t *testing.T) {
	for _, n := range []int{2, 3, 8, 64} {
		for _, y := range []float64{0, 0.999, 1.0} {
			got := quantizeSamplePlaybackSpeed(y, n)
			if got < 0 || got > 1 {
				t.Fatalf("divisions=%d y=%v produced %v, outside 0..1", n, y, got)
			}
		}
	}
}

// The defaults must reproduce the original hardcoded mapping exactly:
// bottom = half speed, middle = normal, top = double.
func TestDefaultSpeedMappingMatchesOriginalBehaviour(t *testing.T) {
	cases := []struct {
		y     float64
		speed float64
	}{
		{0.0, 0.5},
		{0.5, 1.0},
		{1.0, 2.0},
	}
	for _, tc := range cases {
		offset := samplePlaybackSpeedOffset(tc.y, defaultSamplePlaybackSpeedCenter)
		offset *= defaultSamplePlaybackSpeedRange
		bend := speedOffsetToBend(offset)
		if got := bendToSpeed(bend); !closeTo(got, tc.speed, 0.001) {
			t.Fatalf("y=%v gave speed %.4f, want %.4f", tc.y, got, tc.speed)
		}
	}
}

// Halving the range should pull both extremes toward normal speed.
func TestSpeedRangeCompressesTowardNormal(t *testing.T) {
	offset := samplePlaybackSpeedOffset(1.0, 0.5) * 0.5
	top := bendToSpeed(speedOffsetToBend(offset))

	if top >= 2.0 {
		t.Fatalf("top speed %.3f at half range, want well under 2.0", top)
	}
	if !closeTo(top, math.Sqrt2, 0.01) {
		t.Fatalf("top speed %.3f at half range, want ~1.414 (+6 semitones)", top)
	}
}

func TestZeroSpeedRangePinsToNormal(t *testing.T) {
	for _, y := range []float64{0, 0.25, 0.5, 0.75, 1} {
		offset := samplePlaybackSpeedOffset(y, 0.5) * 0.0
		bend := speedOffsetToBend(offset)
		if got := bendToSpeed(bend); !closeTo(got, 1.0, 1e-6) {
			t.Fatalf("y=%v gave speed %v at zero range, want 1.0", y, got)
		}
	}
}
