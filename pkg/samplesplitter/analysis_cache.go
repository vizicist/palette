package samplesplitter

import (
	"fmt"
	"os"
	"sync"
)

// analysisCache memoizes AnalyzeFile results. Analyzing one MP3 spawns an
// ffmpeg process to convert it to WAV, so without this a directory of samples
// is re-converted from scratch every time a preset touches a sample parameter
// - which, during a quad load, happens many times over.
//
// Entries are keyed on the file's identity (path, size, modification time) plus
// every analyze option that changes the result, so an edited file or a changed
// split mode re-analyzes rather than returning a stale cue.
type analysisCache struct {
	mu      sync.Mutex
	entries map[string]analysisEntry
}

type analysisEntry struct {
	cue      CueData
	waveform []float64
}

// maxAnalysisCacheEntries bounds memory if something repeatedly analyzes
// changing files. A sample library is far smaller than this in practice.
const maxAnalysisCacheEntries = 512

func newAnalysisCache() *analysisCache {
	return &analysisCache{entries: make(map[string]analysisEntry)}
}

func analysisCacheKey(path string, opts AnalyzeOptions) (string, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%s|%d|%d|%s|%g|%g|%g|%d|%g",
		path, info.Size(), info.ModTime().UnixNano(),
		opts.Mode, opts.Interval, opts.SilenceThreshold, opts.SilenceMinimum,
		opts.WordsPerSplit, opts.WordThreshold), true
}

// wrap returns an analyze function that serves repeat requests from the cache.
// Only successful analyses are cached; errors may be transient (a file still
// being written, say) and shouldn't be remembered.
func (c *analysisCache) wrap(analyze analyzeMP3Func) analyzeMP3Func {
	if c == nil {
		return analyze
	}
	return func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		key, ok := analysisCacheKey(path, opts)
		if !ok {
			return analyze(path, opts)
		}

		c.mu.Lock()
		entry, hit := c.entries[key]
		c.mu.Unlock()
		if hit {
			return entry.cue, entry.waveform, nil
		}

		cue, waveform, err := analyze(path, opts)
		if err != nil {
			return cue, waveform, err
		}

		c.mu.Lock()
		if len(c.entries) >= maxAnalysisCacheEntries {
			c.entries = make(map[string]analysisEntry)
		}
		c.entries[key] = analysisEntry{cue: cue, waveform: waveform}
		c.mu.Unlock()
		return cue, waveform, nil
	}
}

func (c *analysisCache) clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries = make(map[string]analysisEntry)
	c.mu.Unlock()
}
