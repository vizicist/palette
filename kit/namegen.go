package kit

// Random, human-memorable, anonymous names built from three words, e.g.
// "gentle_otter_sunrise".  Intended for naming recordings (filenames and,
// eventually, YouTube titles).  The words come from the EFF large wordlist,
// which is curated to be memorable, distinct, and inoffensive.  With 7772
// words, three-word names give about 4.7e11 combinations, so random
// collisions are effectively impossible at Palette scale; UniqueRandomName
// still checks against existing names to guarantee it.
//
// Words are joined with underscores so a name is directly usable as a
// filename.  NameTitle turns a name back into a spaced, capitalized title.

import (
	"crypto/rand"
	_ "embed"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// The EFF large wordlist (https://www.eff.org/dice), CC-BY 3.0,
// with dice indices and hyphenated words removed.
//
//go:embed eff_wordlist.txt
var nameWordsRaw string

var nameWords = strings.Fields(nameWordsRaw)

const (
	nameWordCount = 3
	nameSeparator = "_"
)

// RandomName returns a three-word name like "gentle_otter_sunrise",
// suitable for use as a filename.
func RandomName() string {
	words := make([]string, nameWordCount)
	max := big.NewInt(int64(len(nameWords)))
	for i := range words {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand only fails if the OS entropy source is broken
			LogFatal(fmt.Errorf("RandomName: crypto/rand failed: %w", err))
		}
		words[i] = nameWords[n.Int64()]
	}
	return strings.Join(words, nameSeparator)
}

// UniqueRandomName returns a random three-word name for which isUsed
// reports false.  isUsed is typically a check against existing files or
// a registry of already-assigned names.
func UniqueRandomName(isUsed func(name string) bool) (string, error) {
	// Collisions are astronomically unlikely, so a handful of retries
	// only ever matters if the caller's name space is nearly full.
	for attempt := 0; attempt < 100; attempt++ {
		name := RandomName()
		if !isUsed(name) {
			return name, nil
		}
	}
	return "", fmt.Errorf("UniqueRandomName: unable to find unused name after 100 attempts")
}

var generatedNamePattern = regexp.MustCompile(`^[a-z]+_[a-z]+_[a-z]+$`)

// IsGeneratedName reports whether name (with no extension) has the shape of
// a generated three-word name, e.g. "gentle_otter_sunrise".
func IsGeneratedName(name string) bool {
	return generatedNamePattern.MatchString(name)
}

// NameTitle converts a name to a display title,
// e.g. "gentle_otter_sunrise" -> "Gentle Otter Sunrise".
func NameTitle(name string) string {
	words := strings.Split(name, nameSeparator)
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
