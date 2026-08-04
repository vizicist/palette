package kit

import (
	"math"
	"math/rand"
	"testing"
)

const testW = 128
const testH = 128

func flatFrame(v float64) []float64 {
	luma := make([]float64, testW*testH)
	for i := range luma {
		luma[i] = v
	}
	return luma
}

func checkerFrame(cell int) []float64 {
	luma := make([]float64, testW*testH)
	for y := 0; y < testH; y++ {
		for x := 0; x < testW; x++ {
			if ((x/cell)+(y/cell))%2 == 0 {
				luma[y*testW+x] = 0.9
			} else {
				luma[y*testW+x] = 0.1
			}
		}
	}
	return luma
}

func noisyGradientFrame(r *rand.Rand) []float64 {
	luma := make([]float64, testW*testH)
	for y := 0; y < testH; y++ {
		for x := 0; x < testW; x++ {
			v := float64(x)/testW*0.7 + r.Float64()*0.3
			luma[y*testW+x] = math.Min(1.0, v)
		}
	}
	return luma
}

func TestInterestBlankIsUninteresting(t *testing.T) {
	m := computeInterestMetrics(flatFrame(0.0), testW, testH, nil)
	if m.Score > 0.01 {
		t.Errorf("all-black should score ~0, got %f (metrics %+v)", m.Score, m)
	}
}

func TestInterestWhiteIsUninteresting(t *testing.T) {
	m := computeInterestMetrics(flatFrame(1.0), testW, testH, nil)
	if m.Score > 0.01 {
		t.Errorf("all-white should score ~0, got %f (metrics %+v)", m.Score, m)
	}
	// The failure must come from flatness, not from the blank detector.
	if m.NonBlack < 0.99 {
		t.Errorf("all-white frame miscounted as black: nonblack=%f", m.NonBlack)
	}
}

func TestInterestVariedIsInteresting(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	prev := noisyGradientFrame(r)
	cur := noisyGradientFrame(r)
	m := computeInterestMetrics(cur, testW, testH, prev)
	if m.Score < 0.7 {
		t.Errorf("noisy gradient should score high, got %f (metrics %+v)", m.Score, m)
	}
}

func TestInterestEdgesScoreAboveFlat(t *testing.T) {
	checker := computeInterestMetrics(checkerFrame(8), testW, testH, nil)
	flat := computeInterestMetrics(flatFrame(0.5), testW, testH, nil)
	if checker.Score <= flat.Score {
		t.Errorf("checkerboard %f should out-score flat gray %f", checker.Score, flat.Score)
	}
	if checker.EdgeFrac <= 0 {
		t.Errorf("checkerboard should have edges, got %f", checker.EdgeFrac)
	}
}

func TestInterestStaticPenalized(t *testing.T) {
	frame := checkerFrame(8)
	moving := computeInterestMetrics(checkerFrame(16), testW, testH, frame)
	static := computeInterestMetrics(frame, testW, testH, frame)
	if static.Score >= moving.Score {
		t.Errorf("static frame %f should score below changing frame %f", static.Score, moving.Score)
	}
}

// TestCaptureMonitorSmoke exercises the real capture path when a display
// is available, and skips on headless or non-Windows environments.
func TestCaptureMonitorSmoke(t *testing.T) {
	luma, err := CaptureMonitorLuma(-1, 64, 64)
	if err != nil {
		t.Skipf("no capturable display: %v", err)
	}
	if len(luma) != 64*64 {
		t.Fatalf("expected %d luma values, got %d", 64*64, len(luma))
	}
	for i, v := range luma {
		if v < 0.0 || v > 1.0 {
			t.Fatalf("luma[%d] out of range: %f", i, v)
		}
	}
	// Whatever is on screen, the metrics must come back finite.
	m := computeInterestMetrics(luma, 64, 64, nil)
	if math.IsNaN(m.Score) || m.Score < 0 || m.Score > 1 {
		t.Fatalf("bad score from live capture: %+v", m)
	}
	t.Logf("live capture metrics: %+v", m)
}

// TestListCaptureMonitors verifies the monitor listing on machines with a
// display, and logs it so the capture indexes can be inspected.
func TestListCaptureMonitors(t *testing.T) {
	list, err := ListCaptureMonitors()
	if err != nil {
		t.Skipf("no monitor enumeration: %v", err)
	}
	if len(list) < 2 || list[0] != '[' {
		t.Fatalf("unexpected listing: %s", list)
	}
	t.Logf("capture monitors: %s", list)
}

func TestAutoFeedbackWeighsLess(t *testing.T) {
	setupFeedbackTest(t)

	// The same avoided set, recorded automatically in one category and by
	// button press in another; the auto example must produce a smaller
	// badness. (Categories are independent stores, which makes them a
	// convenient way to compare the two sources in isolation.)
	params := exampleParams("0.05", "5.0")
	if err := AddRandFeedbackSource("cat_auto", "avoid", params, "auto"); err != nil {
		t.Fatal(err)
	}
	if err := AddRandFeedback("cat_user", "avoid", params); err != nil {
		t.Fatal(err)
	}

	autoBadness := badnessOf(t, params, "cat_auto")
	userBadness := badnessOf(t, params, "cat_user")

	if autoBadness >= userBadness {
		t.Errorf("auto example should weigh less: auto=%f user=%f", autoBadness, userBadness)
	}
	if autoBadness <= 0 {
		t.Errorf("auto example should still contribute: auto=%f", autoBadness)
	}
}
