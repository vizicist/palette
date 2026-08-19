package samplesplitter

import (
	"fmt"
	"testing"
	"time"
)

// The decoded-PCM cache is keyed by pathname, so an entry is only usable while
// the file behind it is unchanged. It used to key on the path alone - unlike
// the analysis cache, which has always included size and mtime - so replacing a
// sample left the old audio paired with cues analysed from the new one.
func TestDecodedCacheRejectsAReplacedFile(t *testing.T) {
	a := &AudioManager{cache: map[string]*audioBuffer{}}

	modTime := time.Now()
	a.cache["a.mp3"] = &audioBuffer{samples: []int16{1, 2, 3}, size: 100, modTime: modTime}

	// Same identity: usable.
	if got := a.cache["a.mp3"]; got.size != 100 || !got.modTime.Equal(modTime) {
		t.Fatal("the entry does not carry the identity it was decoded from")
	}

	// A different size, or a different mtime, must not match.
	cached := a.cache["a.mp3"]
	if cached.size == 200 {
		t.Fatal("size did not change")
	}
	if cached.modTime.Equal(modTime.Add(time.Second)) {
		t.Fatal("modTime comparison is not doing anything")
	}
}

// Nothing evicted these entries, and a decoded WAV is far larger than the MP3
// it came from, so a long run that kept swapping files grew until the process
// died.
func TestDecodedCacheIsBounded(t *testing.T) {
	a := &AudioManager{cache: map[string]*audioBuffer{}}

	for i := 0; i < maxDecodedCacheEntries*2; i++ {
		a.cacheClock++
		a.cache[fmt.Sprintf("s%03d.mp3", i)] = &audioBuffer{used: a.cacheClock}
		a.evictDecodedLocked(maxDecodedCacheEntries)
	}

	if len(a.cache) > maxDecodedCacheEntries {
		t.Fatalf("cache holds %d entries, want at most %d", len(a.cache), maxDecodedCacheEntries)
	}
	// The survivors are the most recently used ones.
	if _, ok := a.cache["s000.mp3"]; ok {
		t.Error("the oldest entry survived eviction")
	}
	last := fmt.Sprintf("s%03d.mp3", maxDecodedCacheEntries*2-1)
	if _, ok := a.cache[last]; !ok {
		t.Errorf("the newest entry %s was evicted", last)
	}
}

// Eviction picks the least recently served entry, not an arbitrary one.
func TestDecodedCacheEvictsLeastRecentlyUsed(t *testing.T) {
	a := &AudioManager{cache: map[string]*audioBuffer{}}
	a.cache["old"] = &audioBuffer{used: 1}
	a.cache["middle"] = &audioBuffer{used: 2}
	a.cache["fresh"] = &audioBuffer{used: 3}

	a.evictDecodedLocked(2)

	if _, ok := a.cache["old"]; ok {
		t.Error("evicted something other than the least recently used entry")
	}
	if _, ok := a.cache["fresh"]; !ok {
		t.Error("the most recently used entry was evicted")
	}
}
