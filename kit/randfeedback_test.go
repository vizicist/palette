package kit

import (
	"os"
	"path/filepath"
	"testing"
)

// setupFeedbackTest installs a small synthetic ParamDefs table and points
// the feedback database at a temp directory, restoring everything after.
func setupFeedbackTest(t *testing.T) {
	t.Helper()
	InitLog("")

	oldDefs := ParamDefs
	ParamDefs = map[string]ParamDef{
		"visual.alpha": {
			Category:      "visual",
			TypedParamDef: ParamDefFloat{min: 0, max: 1, randmin: 0, randmax: 1, hasRand: true},
		},
		"visual.size": {
			Category:      "visual",
			TypedParamDef: ParamDefFloat{min: 0, max: 10, randmin: 0, randmax: 10, hasRand: true},
		},
		"visual.count": {
			Category:      "visual",
			TypedParamDef: ParamDefInt{min: 1, max: 100, randmin: 1, randmax: 100, hasRand: true},
		},
		"visual.filled": {
			Category:      "visual",
			TypedParamDef: ParamDefBool{randmax: 0.5, hasRand: true},
		},
		"visual.shape": {
			Category:      "visual",
			TypedParamDef: ParamDefString{values: []string{"square", "circle", "star"}, hasRand: true, randmax: ""},
		},
		"sound.level": {
			Category:      "sound",
			TypedParamDef: ParamDefFloat{min: 0, max: 1, randmin: 0, randmax: 1, hasRand: true},
		},
	}

	tmp := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmp)
	t.Setenv("HOME", tmp)

	theFeedbackDB = nil
	t.Cleanup(func() {
		ParamDefs = oldDefs
		theFeedbackDB = nil
	})
}

func exampleParams(alpha, size string) map[string]string {
	return map[string]string{
		"visual.alpha":  alpha,
		"visual.size":   size,
		"visual.count":  "50",
		"visual.filled": "true",
		"visual.shape":  "circle",
	}
}

func badnessOf(t *testing.T, candidate map[string]string, category string) float64 {
	t.Helper()
	examples := feedbackExamplesForCategory(category)
	avoidStats := newFeedbackBinStats(examples, "avoid")
	likeStats := newFeedbackBinStats(examples, "like")
	return feedbackBadness(candidate, examples, avoidStats, likeStats)
}

func TestFeedbackAvoidRaisesBadness(t *testing.T) {
	setupFeedbackTest(t)

	// Several avoided sets share low alpha but differ elsewhere, so the
	// per-param evidence should generalize: low alpha alone scores worse.
	sizes := []string{"1.0", "4.0", "7.0", "9.5"}
	for _, sz := range sizes {
		if err := AddRandFeedback("visual", "avoid", exampleParams("0.02", sz)); err != nil {
			t.Fatal(err)
		}
	}

	low := badnessOf(t, exampleParams("0.05", "5.0"), "visual")
	high := badnessOf(t, exampleParams("0.9", "5.0"), "visual")
	if low <= high {
		t.Errorf("low-alpha candidate should score worse: low=%f high=%f", low, high)
	}
}

func TestFeedbackLikeLowersBadness(t *testing.T) {
	setupFeedbackTest(t)

	for _, sz := range []string{"2.0", "6.0", "8.0"} {
		if err := AddRandFeedback("visual", "like", exampleParams("0.9", sz)); err != nil {
			t.Fatal(err)
		}
	}

	liked := badnessOf(t, exampleParams("0.85", "5.0"), "visual")
	other := badnessOf(t, exampleParams("0.1", "5.0"), "visual")
	if liked >= other {
		t.Errorf("liked-alpha candidate should score better: liked=%f other=%f", liked, other)
	}
}

func TestFeedbackKernelCatchesWholeSet(t *testing.T) {
	setupFeedbackTest(t)

	// One avoided set: the kernel should penalize a near-duplicate more
	// than a set that is far away in every parameter.
	avoided := exampleParams("0.5", "5.0")
	if err := AddRandFeedback("visual", "avoid", avoided); err != nil {
		t.Fatal(err)
	}

	near := exampleParams("0.52", "5.1")
	far := map[string]string{
		"visual.alpha":  "0.95",
		"visual.size":   "0.5",
		"visual.count":  "5",
		"visual.filled": "false",
		"visual.shape":  "star",
	}
	if b1, b2 := badnessOf(t, near, "visual"), badnessOf(t, far, "visual"); b1 <= b2 {
		t.Errorf("near-duplicate should score worse: near=%f far=%f", b1, b2)
	}
}

func TestFeedbackCategoriesIndependent(t *testing.T) {
	setupFeedbackTest(t)

	if err := AddRandFeedback("visual", "avoid", exampleParams("0.02", "5.0")); err != nil {
		t.Fatal(err)
	}
	// Sound has no feedback, so any sound candidate scores zero.
	b := badnessOf(t, map[string]string{"sound.level": "0.02"}, "sound")
	if b != 0 {
		t.Errorf("sound category should be untrained, got badness %f", b)
	}
}

func TestFeedbackPersistence(t *testing.T) {
	setupFeedbackTest(t)

	if err := AddRandFeedback("visual", "avoid", exampleParams("0.02", "5.0")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(LocalPaletteDir(), "config", "randfeedback.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("feedback db not written: %v", err)
	}

	// Drop the in-memory copy and reload from disk.
	theFeedbackDB = nil
	examples := feedbackExamplesForCategory("visual")
	if len(examples) != 1 {
		t.Fatalf("expected 1 example after reload, got %d", len(examples))
	}
	if examples[0].Verdict != "avoid" || examples[0].Params["visual.alpha"] != "0.02" {
		t.Errorf("reloaded example mismatch: %+v", examples[0])
	}
}

func TestFeedbackRejectsBadVerdict(t *testing.T) {
	setupFeedbackTest(t)
	if err := AddRandFeedback("visual", "dislike", exampleParams("0.5", "5.0")); err == nil {
		t.Error("expected error for unknown verdict")
	}
	if err := AddRandFeedback("visual", "avoid", map[string]string{}); err == nil {
		t.Error("expected error for empty params")
	}
}

func TestPickWithoutFeedbackIsPlainRandom(t *testing.T) {
	setupFeedbackTest(t)

	picked := PickRandomParamsForCategory("visual")
	for _, name := range []string{"visual.alpha", "visual.size", "visual.count", "visual.filled", "visual.shape"} {
		if _, ok := picked[name]; !ok {
			t.Errorf("missing %s in picked params", name)
		}
	}
}

func TestPickAvoidsTrainedRegion(t *testing.T) {
	setupFeedbackTest(t)

	// Teach it that low alpha is bad, several times.
	for _, sz := range []string{"1.0", "3.0", "5.0", "7.0", "9.0"} {
		for _, a := range []string{"0.01", "0.05", "0.1"} {
			if err := AddRandFeedback("visual", "avoid", exampleParams(a, sz)); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Statistically, picks should now land in the lowest alpha bin much
	// less often than the uniform 1-in-6.
	const trials = 500
	lowCount := 0
	for i := 0; i < trials; i++ {
		picked := PickRandomParamsForCategory("visual")
		alpha, err := ParseFloat(picked["visual.alpha"], "test")
		if err != nil {
			t.Fatal(err)
		}
		if alpha < 1.0/6.0 {
			lowCount++
		}
	}
	frac := float64(lowCount) / trials
	if frac > 0.13 {
		t.Errorf("low-alpha picks not suppressed: %.3f of picks (uniform would be 0.167)", frac)
	}
}
