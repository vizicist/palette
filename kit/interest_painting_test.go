package kit

import (
	"math"
	"math/rand"
	"testing"
)

func paintTestDot(frame []float64, pos CursorPos, value float64) {
	for y := 0; y < testH; y++ {
		for x := 0; x < testW; x++ {
			if math.Hypot((float64(x)+0.5)/testW-pos.X, (float64(y)+0.5)/testH-pos.Y) < 0.025 {
				frame[y*testW+x] = value
			}
		}
	}
}

func TestPaintingFollowsGestureLocations(t *testing.T) {
	transform := interestCursorTransform{scaleX: 1, scaleY: 1}
	cases := []struct {
		name     string
		place    func(CursorPos) CursorPos
		wantPass bool
	}{
		{"at cursor", func(p CursorPos) CursorPos { return p }, true},
		{"fixed center", func(CursorPos) CursorPos { return CursorPos{X: 0.5, Y: 0.5} }, false},
		{"wrong vertical location", func(p CursorPos) CursorPos { p.Y = 1 - p.Y; return p }, false},
		{"ignores vertical input", func(p CursorPos) CursorPos { p.Y = 0.5; return p }, false},
		{"opposite corner", func(p CursorPos) CursorPos { p.X, p.Y = 1-p.X, 1-p.Y; return p }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			total, count := 0.0, 0
			for _, stroke := range interestPaintingStrokes() {
				before := flatFrame(0)
				previous := transform.screenPosition(stroke[0])
				for _, pos := range stroke {
					expected := transform.screenPosition(pos)
					after := append([]float64(nil), before...)
					paintTestDot(after, tc.place(expected), 0.9)
					painting := computePaintingMetrics(before, after, nil, testW, testH, previous, expected)
					appearance := computeInterestMetrics(after, testW, testH, before)
					total += paintingInterestScore(appearance.Score, painting)
					count++
					previous, before = expected, after
				}
			}
			score := total / float64(count)
			if tc.wantPass && score <= interestLevelDefault {
				t.Errorf("sparse painting must pass without fullscreen texture: score %f", score)
			}
			if !tc.wantPass && score >= 0.5 {
				t.Errorf("misplaced output must not pass: score %f", score)
			}
		})
	}
}

func TestPaintingRejectsUnrelatedOutput(t *testing.T) {
	pos := CursorPos{X: 0.2, Y: 0.8}
	r := rand.New(rand.NewSource(41))
	for _, tc := range []struct {
		name                string
		before, after, idle []float64
	}{
		{"blank", flatFrame(0), flatFrame(0), nil},
		{"static detailed image", checkerFrame(8), checkerFrame(8), nil},
		{"fullscreen flash", flatFrame(0), flatFrame(1), nil},
		{"busy random animation", noisyGradientFrame(r), noisyGradientFrame(r), nil},
		{"continuing background motion", flatFrame(0.2), flatFrame(0.32), flatFrame(0.12)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			painting := computePaintingMetrics(tc.before, tc.after, tc.idle, testW, testH, pos, pos)
			appearance := computeInterestMetrics(tc.after, testW, testH, tc.before)
			if score := paintingInterestScore(appearance.Score, painting); score >= 0.1 {
				t.Errorf("unrelated animation must score near zero, got %f (%+v)", score, painting)
			}
		})
	}
}

func TestPaintingCountsDarkMarks(t *testing.T) {
	pos := CursorPos{X: 0.8, Y: 0.2}
	before, after := flatFrame(1), flatFrame(1)
	paintTestDot(after, pos, 0.1)
	m := computePaintingMetrics(before, after, nil, testW, testH, pos, pos)
	if m.Score < 0.95 {
		t.Errorf("dark paint on white should track just like light paint: %+v", m)
	}
}

func TestPaintingDiscountsFadingEarlierMarks(t *testing.T) {
	pos := CursorPos{X: 0.8, Y: 0.8}
	before, after, idle := flatFrame(0.3), flatFrame(0.2), flatFrame(0.1)
	paintTestDot(after, pos, 0.9)
	m := computePaintingMetrics(before, after, idle, testW, testH, pos, pos)
	if m.Score < 0.95 {
		t.Errorf("fading history should not hide new localized paint: %+v", m)
	}
}

func TestPaintingRequiresVisibleResponse(t *testing.T) {
	pos := CursorPos{X: 0.2, Y: 0.2}
	before, after := flatFrame(0), flatFrame(0)
	paintTestDot(after, pos, 0.005)
	if m := computePaintingMetrics(before, after, nil, testW, testH, pos, pos); m.Score != 0 {
		t.Errorf("nearly invisible marks should not pass: %+v", m)
	}
	if m := computePaintingMetrics(nil, after, nil, testW, testH, pos, pos); m.Score != 0 {
		t.Errorf("missing frames should not pass: %+v", m)
	}
}

func TestPaintingCoordinateTransform(t *testing.T) {
	transform := interestCursorTransform{scaleX: 0.8, scaleY: 0.5, offsetX: 0.1, offsetY: 0.2}
	pos := transform.screenPosition(CursorPos{X: 0.2, Y: 0.3})
	if math.Abs(pos.X-0.26) > 1e-9 || math.Abs(pos.Y-0.65) > 1e-9 {
		t.Errorf("capture coordinates must reflect input calibration and OpenGL Y: %+v", pos)
	}
}

func TestInterestingLevelControlsAcceptance(t *testing.T) {
	for _, tc := range []struct {
		score, level float64
		accepted     bool
	}{
		{0.65, 0.5, true},
		{0.65, 0.8, false},
		{0.8, 0.8, true},
		{0, 0, false},
		{0.01, 0, true},
		{0.99, 1, false},
	} {
		if got := acceptsPaintingScore(tc.score, tc.level); got != tc.accepted {
			t.Errorf("score %f at level %f: accepted=%v, want %v", tc.score, tc.level, got, tc.accepted)
		}
	}
}
