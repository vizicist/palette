package kit

// Automatic "interestingness" evaluation of the visual output, used to
// filter random patches by their response to cursor gestures.
//
// After Rand is applied, StartInterestEvaluation draws known strokes while
// sampling the Resolume output. New activity must follow the strokes' positions
// to score well. General image richness is a small bonus, so controllable
// painting is preferred over unrelated, busy animation.

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
	// Measure background animation, then allow the new cursor position to
	// render before capturing its response. Twelve probes take a few seconds.
	interestQuietDelay    = 100 * time.Millisecond
	interestResponseDelay = 300 * time.Millisecond
	// Default minimum painting score for accepting a random patch.
	interestLevelDefault = 0.80
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
	Category   string          `json:"category"`
	Patch      string          `json:"patch"`
	Time       string          `json:"time"`
	Samples    int             `json:"samples"`
	Score      float64         `json:"score"`
	Last       InterestMetrics `json:"last"`
	Appearance float64         `json:"appearance"`
	Painting   PaintingMetrics `json:"painting"`
	Level      float64         `json:"level"`
	Accepted   bool            `json:"accepted"`
	Error      string          `json:"error,omitempty"`
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
// current output for a patch and category: draw controlled strokes, sample
// their spatial response on the output monitor, and report acceptance.
// patch may be "*" to spread the gestures over all four patches (they share
// params after a Rand in all-patches mode).
func StartInterestEvaluation(patch string, category string) error {

	enabled, err := GetParamBool("global.interesteval")
	if err == nil && !enabled {
		return nil
	}

	// Validate the requested patch and category before starting the probes.
	paramPatch := patch
	if paramPatch == "*" {
		paramPatch = "A"
	}
	p, ok := Patchs[paramPatch]
	if !ok {
		return fmt.Errorf("StartInterestEvaluation: no such patch %s", paramPatch)
	}
	hasCategory := false
	prefix := category + "."
	for _, nm := range p.ParamNames() {
		if len(nm) > len(prefix) && nm[:len(prefix)] == prefix {
			hasCategory = true
			break
		}
	}
	if !hasCategory {
		return fmt.Errorf("StartInterestEvaluation: no params for category %s", category)
	}

	gen := interestGeneration.Add(1)
	go runInterestEvaluation(gen, patch, category)
	return nil
}

func runInterestEvaluation(gen int64, patch string, category string) {

	superseded := func() bool { return interestGeneration.Load() != gen }

	monitor := -1
	if v, err := GetParamInt("global.interestmonitor"); err == nil {
		monitor = v
	}

	var last InterestMetrics
	var painting PaintingMetrics
	total, appearance := 0.0, 0.0
	samples := 0
	transform := currentInterestCursorTransform()
	for index, stroke := range interestPaintingStrokes() {
		if superseded() {
			return
		}
		gesturePatch := patch
		if patch == "*" {
			gesturePatch = string("ABCD"[index])
		}
		err := samplePaintingStroke(gesturePatch, stroke, monitor, transform, superseded,
			func(frame InterestMetrics, response PaintingMetrics) {
				last = frame
				appearance += frame.Score
				painting.Response += response.Response
				painting.Localization += response.Localization
				painting.Score += response.Score
				total += paintingInterestScore(frame.Score, response)
				samples++
			})
		if superseded() {
			return
		}
		if err != nil {
			LogWarn("interest evaluation: capture failed", "err", err)
			setInterestResult(&interestResult{
				Category: category, Patch: patch,
				Time: time.Now().Format(time.RFC3339), Error: err.Error(),
			})
			return
		}
	}

	if superseded() || samples == 0 {
		return
	}
	score := total / float64(samples)
	appearance /= float64(samples)
	painting.Response /= float64(samples)
	painting.Localization /= float64(samples)
	painting.Score /= float64(samples)
	level := interestLevelDefault
	if v, err := GetParamFloat("global.interestlevel"); err == nil {
		level = v
	}
	accepted := acceptsPaintingScore(score, level)

	LogInfo("interest evaluation", "patch", patch, "category", category,
		"score", score, "painting", painting.Score, "appearance", appearance,
		"samples", samples, "level", level, "accepted", accepted)

	setInterestResult(&interestResult{
		Category: category, Patch: patch,
		Time:    time.Now().Format(time.RFC3339),
		Samples: samples, Score: score, Last: last,
		Appearance: appearance, Painting: painting,
		Level: level, Accepted: accepted,
	})
}

// Capture a quiet interval and a response interval at each cursor position. A
// stroke always releases its cursor, including on capture failure or when a
// newer evaluation supersedes it, so a canceled Rand cannot leave notes held.
func samplePaintingStroke(patch string, stroke [3]CursorPos, monitor int, transform interestCursorTransform,
	superseded func() bool, sample func(InterestMetrics, PaintingMetrics)) error {

	cm := theCursorManager
	// Start each probe stroke on fresh graphics. Otherwise repeated tests of
	// persistent paint find an already-covered canvas and mistake overpainting
	// the same pixels for a patch that does not respond to the cursor.
	if p := Patchs[patch]; p != nil {
		p.clearGraphics()
	}
	time.Sleep(150 * time.Millisecond)
	gid := cm.UniqueGID()
	tag := patch + ",interest"
	var current CursorPos
	active := false
	defer func() {
		if active {
			cm.ExecuteCursorEvent(NewCursorEvent(gid, tag, "up", current))
		}
	}()
	capture := func() ([]float64, error) {
		return CaptureMonitorLuma(monitor, interestCaptureSize, interestCaptureSize)
	}
	previous := transform.screenPosition(stroke[0])
	for index, pos := range stroke {
		if superseded() {
			return nil
		}
		quiet, err := capture()
		if err != nil {
			return err
		}
		time.Sleep(interestQuietDelay)
		before, err := capture()
		if err != nil {
			return err
		}
		if superseded() {
			return nil
		}
		event := "drag"
		if index == 0 {
			event = "down"
		}
		current, active = pos, true
		cm.ExecuteCursorEvent(NewCursorEvent(gid, tag, event, pos))
		time.Sleep(interestResponseDelay)
		if superseded() {
			return nil
		}
		after, err := capture()
		if err != nil {
			return err
		}
		idle := make([]float64, len(before))
		for i := range idle {
			idle[i] = math.Abs(before[i]-quiet[i]) * float64(interestResponseDelay) / float64(interestQuietDelay)
		}
		expected := transform.screenPosition(pos)
		frame := computeInterestMetrics(after, interestCaptureSize, interestCaptureSize, before)
		response := computePaintingMetrics(before, after, idle, interestCaptureSize, interestCaptureSize, previous, expected)
		LogOfType("cursor", "painting spatial response", "expectedX", expected.X, "expectedY", expected.Y,
			"observedX", response.centerX, "observedY", response.centerY,
			"response", response.Response, "localization", response.Localization)
		sample(frame, response)
		previous = expected
	}
	return nil
}
