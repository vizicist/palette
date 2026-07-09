package kit

import (
	"regexp"
	"testing"
)

func TestNameWordlistLoaded(t *testing.T) {
	// The full EFF list has 7772 usable words; allow pruning words from
	// eff_wordlist.txt without breaking the test, but catch a truncated
	// or missing embed.
	if len(nameWords) < 7000 {
		t.Fatalf("nameWords has only %d words, want at least 7000", len(nameWords))
	}
	alpha := regexp.MustCompile(`^[a-z]+$`)
	for _, w := range nameWords {
		if !alpha.MatchString(w) {
			t.Fatalf("wordlist contains non-alphabetic word %q", w)
		}
	}
}

func TestRandomNameFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-z]+_[a-z]+_[a-z]+$`)
	for i := 0; i < 100; i++ {
		name := RandomName()
		if !pattern.MatchString(name) {
			t.Fatalf("RandomName() = %q, want three underscore-separated lowercase words", name)
		}
	}
}

func TestRandomNamesVary(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		seen[RandomName()] = true
	}
	// 50 draws from ~4.7e11 combinations should essentially never collide.
	if len(seen) < 49 {
		t.Fatalf("50 RandomName() calls produced only %d distinct names", len(seen))
	}
}

func TestUniqueRandomNameRespectsIsUsed(t *testing.T) {
	calls := 0
	name, err := UniqueRandomName(func(name string) bool {
		calls++
		return calls <= 3 // reject the first three candidates
	})
	if err != nil {
		t.Fatalf("UniqueRandomName returned error: %v", err)
	}
	if name == "" || calls != 4 {
		t.Fatalf("UniqueRandomName = %q after %d calls, want a name on the 4th", name, calls)
	}
}

func TestUniqueRandomNameGivesUp(t *testing.T) {
	_, err := UniqueRandomName(func(string) bool { return true })
	if err == nil {
		t.Fatal("UniqueRandomName with always-used names should return an error")
	}
}

func TestIsGeneratedName(t *testing.T) {
	cases := map[string]bool{
		"gentle_otter_sunrise":       true,
		"a_b_c":                      true,
		"gentle_otter":               false,
		"gentle_otter_sunrise_extra": false,
		"Gentle_Otter_Sunrise":       false,
		"2025-01-02 10-11-12":        false, // OBS default timestamp name
		"gentle_otter_sunrise.mp4":   false, // caller must strip the extension
		"":                           false,
	}
	for name, want := range cases {
		if got := IsGeneratedName(name); got != want {
			t.Errorf("IsGeneratedName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestNameTitle(t *testing.T) {
	got := NameTitle("gentle_otter_sunrise")
	if got != "Gentle Otter Sunrise" {
		t.Fatalf("NameTitle = %q, want %q", got, "Gentle Otter Sunrise")
	}
	if got := NameTitle("abacus"); got != "Abacus" {
		t.Fatalf("NameTitle single word = %q, want %q", got, "Abacus")
	}
}
