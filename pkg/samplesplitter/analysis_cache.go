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
	mu       sync.Mutex
	entries  map[string]analysisEntry
	inflight map[string]*analysisCall
}

type analysisEntry struct {
	cue      CueData
	waveform []float64
}

// analysisCall is one analysis that is currently running. Later callers asking
// for the same key wait on done rather than starting an ffmpeg of their own:
// the cache lock cannot be held across the analysis itself, so without this a
// burst of requests for one file - which is exactly what a preset load produces
// - would all miss together and each spawn a process.
type analysisCall struct {
	done     chan struct{}
	cue      CueData
	waveform []float64
	err      error
}

// maxAnalysisCacheEntries bounds memory if something repeatedly analyzes
// changing files. A sample library is far smaller than this in practice.
const maxAnalysisCacheEntries = 512

func newAnalysisCache() *analysisCache {
	return &analysisCache{
		entries:  make(map[string]analysisEntry),
		inflight: make(map[string]*analysisCall),
	}
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
		if entry, hit := c.entries[key]; hit {
			c.mu.Unlock()
			return entry.cue, entry.waveform, nil
		}
		if call, running := c.inflight[key]; running {
			// Somebody else is already analyzing this exact file with these
			// exact options. Wait for their result instead of duplicating it.
			c.mu.Unlock()
			<-call.done
			return call.cue, call.waveform, call.err
		}
		call := &analysisCall{done: make(chan struct{})}
		c.inflight[key] = call
		c.mu.Unlock()

		call.cue, call.waveform, call.err = analyze(path, opts)

		c.mu.Lock()
		delete(c.inflight, key)
		// Only successful analyses are cached; errors may be transient (a file
		// still being written, say) and shouldn't be remembered. Waiters still
		// get the error, they just don't get it from the cache next time.
		if call.err == nil {
			if len(c.entries) >= maxAnalysisCacheEntries {
				c.entries = make(map[string]analysisEntry)
			}
			c.entries[key] = analysisEntry{cue: call.cue, waveform: call.waveform}
		}
		c.mu.Unlock()

		// Released only after the entry is in place, so a waiter that turns
		// straight around and asks again finds it cached.
		close(call.done)
		return call.cue, call.waveform, call.err
	}
}

// clear drops the memoized results. Analyses already running are left alone:
// their waiters are owed an answer, and they re-populate a cache that the next
// caller is free to clear again.
func (c *analysisCache) clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries = make(map[string]analysisEntry)
	c.mu.Unlock()
}
