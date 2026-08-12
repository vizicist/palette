package kit

// Automatic "interestingness" evaluation of the visual output, used to
// train the learned Rand feature (randfeedback.go) without button presses.
//
// After Rand is applied, StartInterestEvaluation generates random cursor
// gestures for a few seconds (Palette only draws when someone plays, so
// synthetic input guarantees there is output to judge) while sampling the
// Resolume output monitor at low rate. Each sample is scored with cheap
// classical image statistics - no output and all-one-color output score
// near zero, edge-rich varied output scores near one - and the aggregate
// decides whether the parameter set that Rand produced gets recorded as
// an automatic "avoid" or "like" example.
//
// Automatic examples are stored with Source "auto" and carry less weight
// than button presses (see feedbackExampleWeight), so human judgment
// stays authoritative.

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Size of the downscaled capture the metrics run on.
	interestCaptureSize = 128
	// How long to generate gestures and sample the output.
	interestEvalSeconds = 2.5
	// Delay before the first sample, letting sprites appear.
	interestWarmupSeconds = 0.75
	// Time between samples.
	interestSampleSeconds = 0.5
	// Duration and step count of each generated gesture.
	interestGestureSeconds = 0.8
	interestGestureSteps   = 12
	// Defaults for the auto-label thresholds, used when the
	// global.interestavoid / global.interestlike params are unavailable.
	// An aggregate score below the avoid threshold records an automatic
	// "avoid"; above the like threshold an automatic "like"; the band in
	// between records nothing.
	interestAvoidDefault = 0.15
	interestLikeDefault  = 0.80
)

// InterestMetrics holds the sub-metrics and combined score of one sample.
type InterestMetrics struct {
	Mean     float64 `json:"mean"`     // average luma
	Stddev   float64 `json:"stddev"`   // luma spread; near 0 for blank or all-white
	EdgeFrac float64 `json:"edgefrac"` // fraction of pixels on a strong gradient
	Entropy  float64 `json:"entropy"`  // luma histogram entropy, normalized 0..1
	NonBlack float64 `json:"nonblack"` // fraction of pixels above near-black
	Temporal float64 `json:"temporal"` // mean abs luma change since last sample
	Score    float64 `json:"score"`    // combined 0..1 interestingness
}

// computeInterestMetrics scores one luma frame (values 0..1, w*h pixels).
// prev is the previous sample's luma for the temporal term, or nil.
func computeInterestMetrics(luma []float64, w int, h int, prev []float64) InterestMetrics {

	m := InterestMetrics{}
	n := float64(len(luma))
	if n == 0 {
		return m
	}

	// Mean and stddev.
	sum := 0.0
	for _, v := range luma {
		sum += v
	}
	m.Mean = sum / n
	varsum := 0.0
	for _, v := range luma {
		d := v - m.Mean
		varsum += d * d
	}
	m.Stddev = math.Sqrt(varsum / n)

	// Edge fraction via Sobel gradient magnitude.
	edges := 0
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			i := y*w + x
			gx := luma[i-w+1] + 2*luma[i+1] + luma[i+w+1] -
				luma[i-w-1] - 2*luma[i-1] - luma[i+w-1]
			gy := luma[i+w-1] + 2*luma[i+w] + luma[i+w+1] -
				luma[i-w-1] - 2*luma[i-w] - luma[i-w+1]
			if math.Sqrt(gx*gx+gy*gy) > 0.4 {
				edges++
			}
		}
	}
	interior := float64((w - 2) * (h - 2))
	if interior > 0 {
		m.EdgeFrac = float64(edges) / interior
	}

	// Luma histogram entropy, normalized to 0..1 by the maximum possible.
	const bins = 32
	var hist [bins]float64
	for _, v := range luma {
		b := int(v * bins)
		if b >= bins {
			b = bins - 1
		}
		hist[b]++
	}
	entropy := 0.0
	for _, c := range hist {
		if c > 0 {
			p := c / n
			entropy -= p * math.Log2(p)
		}
	}
	m.Entropy = entropy / math.Log2(bins)

	// Fraction of pixels that aren't near-black.
	nonblack := 0
	for _, v := range luma {
		if v > 0.06 {
			nonblack++
		}
	}
	m.NonBlack = float64(nonblack) / n

	// Mean abs change from the previous sample.
	if prev != nil && len(prev) == len(luma) {
		diff := 0.0
		for i, v := range luma {
			diff += math.Abs(v - prev[i])
		}
		m.Temporal = diff / n
	}

	// Combine multiplicatively over saturating curves so any catastrophic
	// failure (blank, all-white, no edges) zeroes the score, while
	// good-enough metrics don't over-reward.
	sat := func(x, target float64) float64 {
		if x >= target {
			return 1.0
		}
		return x / target
	}
	m.Score = sat(m.Stddev, 0.10) *
		sat(m.EdgeFrac, 0.03) *
		sat(m.Entropy, 0.45) *
		sat(m.NonBlack, 0.08)
	if prev != nil {
		// A completely static image is half as interesting.
		m.Score *= 0.5 + 0.5*sat(m.Temporal, 0.005)
	}
	return m
}

// The most recent evaluation result, readable via the global.interest_score
// API.
type interestResult struct {
	Category string          `json:"category"`
	Patch    string          `json:"patch"`
	Time     string          `json:"time"`
	Samples  int             `json:"samples"`
	Score    float64         `json:"score"`
	Verdict  string          `json:"verdict"` // avoid, like, or none
	Last     InterestMetrics `json:"last"`
	Error    string          `json:"error,omitempty"`
}

var (
	interestMutex      sync.Mutex
	lastInterestResult *interestResult
	// Bumped on every new evaluation; an in-flight one abandons itself
	// when it notices it has been superseded.
	interestGeneration atomic.Int64
)

// LastInterestScoreJSON returns the most recent evaluation as JSON.
func LastInterestScoreJSON() (string, error) {
	interestMutex.Lock()
	defer interestMutex.Unlock()
	if lastInterestResult == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(lastInterestResult)
	if err != nil {
		return "", fmt.Errorf("LastInterestScoreJSON: %w", err)
	}
	return string(bytes), nil
}

func setInterestResult(r *interestResult) {
	interestMutex.Lock()
	lastInterestResult = r
	interestMutex.Unlock()
}

// StartInterestEvaluation kicks off an asynchronous evaluation of the
// current output for a patch and category: generate random gestures for a
// few seconds, sample the output monitor, score it, and feed the verdict
// back into the Rand feedback database. patch may be "*" to spread the
// gestures over all four patches (they share params after a Rand in
// all-patches mode).
func StartInterestEvaluation(patch string, category string) error {

	enabled, err := GetParamBool("global.interesteval")
	if err == nil && !enabled {
		return nil
	}

	// Snapshot the params being judged NOW; the evaluation labels this
	// snapshot even if the user changes params while it runs.
	paramPatch := patch
	if paramPatch == "*" {
		paramPatch = "A"
	}
	p, ok := Patchs[paramPatch]
	if !ok {
		return fmt.Errorf("StartInterestEvaluation: no such patch %s", paramPatch)
	}
	snapshot := map[string]string{}
	prefix := category + "."
	for _, nm := range p.ParamNames() {
		if len(nm) > len(prefix) && nm[:len(prefix)] == prefix {
			snapshot[nm] = p.Get(nm)
		}
	}
	if len(snapshot) == 0 {
		return fmt.Errorf("StartInterestEvaluation: no params for category %s", category)
	}

	gen := interestGeneration.Add(1)
	go runInterestEvaluation(gen, patch, category, snapshot)
	return nil
}

func runInterestEvaluation(gen int64, patch string, category string, snapshot map[string]string) {

	superseded := func() bool { return interestGeneration.Load() != gen }

	// Gesture generator: synthetic input so there is something to judge.
	go func() {
		deadline := time.Now().Add(time.Duration(interestEvalSeconds * float64(time.Second)))
		for time.Now().Before(deadline) && !superseded() {
			gesturePatch := patch
			if patch == "*" {
				gesturePatch = string("ABCD"[time.Now().UnixNano()%4])
			}
			theCursorManager.GenerateRandomGesture(gesturePatch+",interest",
				interestGestureSteps,
				time.Duration(interestGestureSeconds*float64(time.Second)))
		}
	}()

	time.Sleep(time.Duration(interestWarmupSeconds * float64(time.Second)))

	monitor := -1
	if v, err := GetParamInt("global.interestmonitor"); err == nil {
		monitor = v
	}

	var prev []float64
	var last InterestMetrics
	total := 0.0
	samples := 0
	deadline := time.Now().Add(time.Duration((interestEvalSeconds - interestWarmupSeconds) * float64(time.Second)))
	for time.Now().Before(deadline) {
		if superseded() {
			return
		}
		luma, err := CaptureMonitorLuma(monitor, interestCaptureSize, interestCaptureSize)
		if err != nil {
			LogWarn("interest evaluation: capture failed", "err", err)
			setInterestResult(&interestResult{
				Category: category, Patch: patch,
				Time: time.Now().Format(time.RFC3339), Error: err.Error(),
			})
			return
		}
		last = computeInterestMetrics(luma, interestCaptureSize, interestCaptureSize, prev)
		prev = luma
		total += last.Score
		samples++
		time.Sleep(time.Duration(interestSampleSeconds * float64(time.Second)))
	}

	if superseded() || samples == 0 {
		return
	}
	score := total / float64(samples)

	// Thresholds are user-tunable global params; avoid is checked first,
	// so a score of exactly 0 with an avoid threshold of 0 records
	// nothing (0 disables avoid, 1 disables like).
	avoidThreshold := interestAvoidDefault
	if v, err := GetParamFloat("global.interestavoid"); err == nil {
		avoidThreshold = v
	}
	likeThreshold := interestLikeDefault
	if v, err := GetParamFloat("global.interestlike"); err == nil {
		likeThreshold = v
	}

	verdict := "none"
	if score < avoidThreshold {
		verdict = "avoid"
	} else if score > likeThreshold {
		verdict = "like"
	}

	LogInfo("interest evaluation", "patch", patch, "category", category,
		"score", score, "samples", samples, "verdict", verdict)

	setInterestResult(&interestResult{
		Category: category, Patch: patch,
		Time:    time.Now().Format(time.RFC3339),
		Samples: samples, Score: score, Verdict: verdict, Last: last,
	})

	if verdict != "none" && !superseded() {
		if err := AddRandFeedbackSource(category, verdict, snapshot, "auto"); err != nil {
			LogWarn("interest evaluation: feedback failed", "err", err)
		}
	}
}
