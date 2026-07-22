package kit

// Learned feedback for the Rand feature. The Like and Avoid buttons in the
// GUI record the current parameter set of a category as a positive or
// negative example. When Rand is pressed, several candidate parameter sets
// are generated and one is chosen with a probability that favors candidates
// resembling liked examples and disfavors those resembling avoided ones.
//
// The avoidance is learned, not absolute: scoring only tilts a softmax
// choice among candidates, so no region of parameter space is ever
// completely forbidden, and a single stray example has only a mild effect.
//
// Two complementary scoring signals are combined:
//
//  1. Per-parameter evidence: each parameter's range is divided into bins,
//     and the examples give per-bin counts for "avoid" and "like". A
//     candidate scores the summed log-likelihood ratio of its bins under
//     the avoid distribution vs the like distribution (each Laplace-
//     smoothed, falling back to uniform when a side has no examples).
//     This generalizes from few examples: three avoided sets that share
//     alphainitial=0 teach "low alphainitial is bad" regardless of what
//     the other parameters were doing.
//
//  2. Whole-set kernel proximity: a candidate near an avoided example
//     (small normalized distance across all parameters jointly) is
//     penalized, and near a liked example rewarded. This catches
//     combinations whose badness lies in the interaction rather than in
//     any single parameter.
//
// Training is independent per category (misc, sound, visual, effect):
// examples are stored and scored only within their own category.
//
// The database is raw examples in a JSON file; scores are derived fresh
// from it, so the scoring math can change without invalidating anything
// the user has taught it.

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// How many candidate sets Rand draws before choosing one.
	feedbackCandidates = 12
	// Bins per numeric parameter for the per-parameter evidence.
	feedbackBins = 6
	// Softmax temperature: lower = follows the learned scores more
	// strictly, higher = closer to plain uniform random.
	feedbackTemperature = 0.5
	// Weight of the kernel signal relative to the per-parameter signal.
	feedbackKernelWeight = 2.0
	// Kernel width, in mean-squared-normalized-distance units. Small
	// values make the kernel react only to close neighbors of an example.
	feedbackKernelWidth2 = 0.15
	// Newest examples kept per category and verdict. Deliberately far
	// beyond what human button presses will reach - truncation forgets
	// specific combinations the kernel signal remembers, so this is only
	// a backstop against a future automatic labeler running away, not a
	// working limit. Scoring cost stays trivial at this size (a linear
	// scan per Rand press).
	feedbackMaxExamples = 5000
)

type FeedbackExample struct {
	Verdict string            `json:"verdict"` // "avoid" or "like"
	Time    string            `json:"time"`
	Params  map[string]string `json:"params"`
}

type feedbackDB struct {
	// Keyed by category (misc, sound, visual, effect).
	Categories map[string][]FeedbackExample `json:"categories"`
}

var (
	theFeedbackDB     *feedbackDB
	feedbackDBMutex   sync.Mutex
	feedbackRandMutex sync.Mutex
)

func feedbackFilePath() string {
	return filepath.Join(LocalPaletteDir(), "config", "randfeedback.json")
}

// loadFeedbackDB returns the in-memory database, reading it from disk on
// first use. Callers must hold feedbackDBMutex.
func loadFeedbackDB() *feedbackDB {
	if theFeedbackDB != nil {
		return theFeedbackDB
	}
	db := &feedbackDB{Categories: map[string][]FeedbackExample{}}
	bytes, err := os.ReadFile(feedbackFilePath())
	if err == nil {
		if jerr := json.Unmarshal(bytes, db); jerr != nil {
			LogWarn("loadFeedbackDB: unable to parse, starting empty", "path", feedbackFilePath(), "err", jerr)
			db = &feedbackDB{Categories: map[string][]FeedbackExample{}}
		}
	}
	if db.Categories == nil {
		db.Categories = map[string][]FeedbackExample{}
	}
	theFeedbackDB = db
	return db
}

func saveFeedbackDB(db *feedbackDB) error {
	path := feedbackFilePath()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return fmt.Errorf("saveFeedbackDB: %w", err)
	}
	bytes, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("saveFeedbackDB: %w", err)
	}
	return os.WriteFile(path, bytes, 0644)
}

// AddRandFeedback records the given parameter set as a liked or avoided
// example for a category.
func AddRandFeedback(category string, verdict string, params map[string]string) error {
	if verdict != "avoid" && verdict != "like" {
		return fmt.Errorf("AddRandFeedback: verdict must be avoid or like, got %s", verdict)
	}
	if len(params) == 0 {
		return fmt.Errorf("AddRandFeedback: no params for category %s", category)
	}

	feedbackDBMutex.Lock()
	defer feedbackDBMutex.Unlock()

	db := loadFeedbackDB()
	db.Categories[category] = append(db.Categories[category], FeedbackExample{
		Verdict: verdict,
		Time:    time.Now().Format(time.RFC3339),
		Params:  params,
	})

	// Cap each verdict separately so a burst of one kind can't age out
	// all examples of the other.
	examples := db.Categories[category]
	counts := map[string]int{}
	for _, ex := range examples {
		counts[ex.Verdict]++
	}
	if counts["avoid"] > feedbackMaxExamples || counts["like"] > feedbackMaxExamples {
		kept := make([]FeedbackExample, 0, len(examples))
		// Walk newest-first, keep up to the cap of each verdict, then
		// restore original order.
		seen := map[string]int{}
		for i := len(examples) - 1; i >= 0; i-- {
			ex := examples[i]
			if seen[ex.Verdict] < feedbackMaxExamples {
				seen[ex.Verdict]++
				kept = append(kept, ex)
			}
		}
		for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
			kept[i], kept[j] = kept[j], kept[i]
		}
		db.Categories[category] = kept
	}

	LogInfo("AddRandFeedback", "category", category, "verdict", verdict,
		"examples", len(db.Categories[category]))
	return saveFeedbackDB(db)
}

// feedbackExamplesForCategory returns a copy of the category's examples.
func feedbackExamplesForCategory(category string) []FeedbackExample {
	feedbackDBMutex.Lock()
	defer feedbackDBMutex.Unlock()
	db := loadFeedbackDB()
	// NOTE: append instead of the copy builtin, which is shadowed by this
	// package's copy() in copy.go.
	return append([]FeedbackExample(nil), db.Categories[category]...)
}

// feedbackBinOf maps a parameter value into a bin index. Numeric params bin
// their full min..max range, bools get two bins, and enum strings one bin
// per value. Returns ok=false for values that can't be placed.
func feedbackBinOf(def ParamDef, val string) (bin int, nbins int, ok bool) {
	switch td := def.TypedParamDef.(type) {
	case ParamDefFloat:
		f, err := ParseFloat(val, "feedbackBinOf")
		if err != nil || td.max <= td.min {
			return 0, 0, false
		}
		frac := (f - td.min) / (td.max - td.min)
		bin = int(frac * feedbackBins)
		if bin < 0 {
			bin = 0
		}
		if bin >= feedbackBins {
			bin = feedbackBins - 1
		}
		return bin, feedbackBins, true
	case ParamDefInt:
		i, err := ParseInt(val, "feedbackBinOf")
		if err != nil || td.max <= td.min {
			return 0, 0, false
		}
		frac := float64(i-td.min) / float64(td.max-td.min)
		bin = int(frac * feedbackBins)
		if bin < 0 {
			bin = 0
		}
		if bin >= feedbackBins {
			bin = feedbackBins - 1
		}
		return bin, feedbackBins, true
	case ParamDefBool:
		if val == "true" {
			return 1, 2, true
		}
		return 0, 2, true
	case ParamDefString:
		for i, v := range td.values {
			if v == val {
				return i, len(td.values), true
			}
		}
		return 0, 0, false
	}
	return 0, 0, false
}

// feedbackParamDistance returns a normalized 0..1 distance between two
// values of one parameter, for the kernel signal.
func feedbackParamDistance(def ParamDef, a string, b string) (float64, bool) {
	switch td := def.TypedParamDef.(type) {
	case ParamDefFloat:
		fa, erra := ParseFloat(a, "feedbackParamDistance")
		fb, errb := ParseFloat(b, "feedbackParamDistance")
		if erra != nil || errb != nil || td.max <= td.min {
			return 0, false
		}
		d := math.Abs(fa-fb) / (td.max - td.min)
		if d > 1 {
			d = 1
		}
		return d, true
	case ParamDefInt:
		ia, erra := ParseInt(a, "feedbackParamDistance")
		ib, errb := ParseInt(b, "feedbackParamDistance")
		if erra != nil || errb != nil || td.max <= td.min {
			return 0, false
		}
		d := math.Abs(float64(ia-ib)) / float64(td.max-td.min)
		if d > 1 {
			d = 1
		}
		return d, true
	case ParamDefBool, ParamDefString:
		if a == b {
			return 0, true
		}
		return 1, true
	}
	return 0, false
}

// feedbackBinStats holds the per-parameter bin counts derived from one
// category's examples, computed once per Rand press.
type feedbackBinStats struct {
	counts map[string][]float64 // param name -> per-bin counts
	total  map[string]float64   // param name -> number of counted examples
}

func newFeedbackBinStats(examples []FeedbackExample, verdict string) feedbackBinStats {
	st := feedbackBinStats{
		counts: map[string][]float64{},
		total:  map[string]float64{},
	}
	for _, ex := range examples {
		if ex.Verdict != verdict {
			continue
		}
		for name, val := range ex.Params {
			def, hasDef := ParamDefs[name]
			if !hasDef {
				continue
			}
			bin, nbins, ok := feedbackBinOf(def, val)
			if !ok {
				continue
			}
			if st.counts[name] == nil {
				st.counts[name] = make([]float64, nbins)
			}
			if bin < len(st.counts[name]) {
				st.counts[name][bin]++
				st.total[name]++
			}
		}
	}
	return st
}

// logProb returns the Laplace-smoothed log probability of a bin under this
// verdict's distribution for one parameter, or the uniform log probability
// when there are no examples for it.
func (st feedbackBinStats) logProb(name string, bin int, nbins int) float64 {
	counts := st.counts[name]
	total := st.total[name]
	c := 0.0
	if counts != nil && bin < len(counts) {
		c = counts[bin]
	}
	return math.Log((c + 1.0) / (total + float64(nbins)))
}

// feedbackKernelSim returns the similarity (0..1) of a candidate to the
// closest example with the given verdict.
func feedbackKernelSim(candidate map[string]string, examples []FeedbackExample, verdict string) float64 {
	best := 0.0
	for _, ex := range examples {
		if ex.Verdict != verdict {
			continue
		}
		sum := 0.0
		n := 0
		for name, cval := range candidate {
			eval, has := ex.Params[name]
			if !has {
				continue
			}
			def, hasDef := ParamDefs[name]
			if !hasDef {
				continue
			}
			d, ok := feedbackParamDistance(def, cval, eval)
			if !ok {
				continue
			}
			sum += d * d
			n++
		}
		if n == 0 {
			continue
		}
		sim := math.Exp(-(sum / float64(n)) / feedbackKernelWidth2)
		if sim > best {
			best = sim
		}
	}
	return best
}

// feedbackBadness scores one candidate: higher = more like avoided
// examples, lower = more like liked ones. Zero when there is no feedback.
func feedbackBadness(candidate map[string]string, examples []FeedbackExample, avoidStats, likeStats feedbackBinStats) float64 {

	// Signal 1: summed per-parameter log-likelihood ratio.
	llr := 0.0
	for name, val := range candidate {
		def, hasDef := ParamDefs[name]
		if !hasDef {
			continue
		}
		bin, nbins, ok := feedbackBinOf(def, val)
		if !ok {
			continue
		}
		llr += avoidStats.logProb(name, bin, nbins) - likeStats.logProb(name, bin, nbins)
	}

	// Signal 2: kernel proximity to whole examples.
	kernel := feedbackKernelSim(candidate, examples, "avoid") -
		feedbackKernelSim(candidate, examples, "like")

	return llr + feedbackKernelWeight*kernel
}

// randomParamsOnce generates one plain random parameter set for a category,
// the same distribution the Rand button always used.
func randomParamsOnce(category string) map[string]string {
	m := map[string]string{}
	for name, def := range ParamDefs {
		if def.Category == category || category == "*" {
			if v := RandomValueForParam(def); v != "" {
				m[name] = v
			}
		}
	}
	return m
}

// PickRandomParamsForCategory generates several candidate random parameter
// sets and picks one, softmax-weighted by the learned feedback. With no
// feedback recorded this reduces to a single plain random draw.
func PickRandomParamsForCategory(category string) map[string]string {

	examples := feedbackExamplesForCategory(category)
	if len(examples) == 0 {
		return randomParamsOnce(category)
	}

	avoidStats := newFeedbackBinStats(examples, "avoid")
	likeStats := newFeedbackBinStats(examples, "like")

	candidates := make([]map[string]string, feedbackCandidates)
	badness := make([]float64, feedbackCandidates)
	minBadness := math.Inf(1)
	for i := range candidates {
		candidates[i] = randomParamsOnce(category)
		badness[i] = feedbackBadness(candidates[i], examples, avoidStats, likeStats)
		if badness[i] < minBadness {
			minBadness = badness[i]
		}
	}

	// Softmax over -badness/T, stabilized by the minimum.
	weights := make([]float64, feedbackCandidates)
	totalWeight := 0.0
	for i, b := range badness {
		weights[i] = math.Exp(-(b - minBadness) / feedbackTemperature)
		totalWeight += weights[i]
	}

	feedbackRandMutex.Lock()
	r := rand.Float64() * totalWeight
	feedbackRandMutex.Unlock()
	for i, w := range weights {
		r -= w
		if r <= 0 {
			return candidates[i]
		}
	}
	return candidates[feedbackCandidates-1]
}
