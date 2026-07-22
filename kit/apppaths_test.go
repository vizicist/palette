package kit

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFirstExistingPathSkipsMissingAndPicksHighestGlobMatch(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Bidule 0.9776 P64", "Bidule 0.9780 P64"} {
		bundle := filepath.Join(dir, name+".app", "Contents", "MacOS")
		if err := os.MkdirAll(bundle, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bundle, "Bidule P64"), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got := firstExistingPath([]string{
		filepath.Join(dir, "does-not-exist"),
		filepath.Join(dir, "Bidule*.app", "Contents", "MacOS", "Bidule*"),
	})
	want := filepath.Join(dir, "Bidule 0.9780 P64.app", "Contents", "MacOS", "Bidule P64")
	if got != want {
		t.Fatalf("firstExistingPath() = %q, want %q", got, want)
	}
}

func TestFirstExistingPathReturnsEmptyWhenNothingMatches(t *testing.T) {
	dir := t.TempDir()
	if got := firstExistingPath([]string{"", filepath.Join(dir, "nope*")}); got != "" {
		t.Fatalf("firstExistingPath() = %q, want empty", got)
	}
}

func TestObsConfigDirIsPlatformCorrect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got := ObsConfigDir()
	var want string
	switch runtime.GOOS {
	case "darwin":
		want = filepath.Join(home, "Library", "Application Support", "obs-studio")
	case "windows":
		want = filepath.Join(home, "obs-studio")
	default:
		want = filepath.Join(home, ".config", "obs-studio")
	}
	if got != want {
		t.Fatalf("ObsConfigDir() = %q, want %q", got, want)
	}
}

func TestCandidatePathsAreNotWindowsOnlyOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-specific expectations")
	}
	for name, candidates := range map[string][]string{
		"bidule": BiduleCandidatePaths(),
		"obs":    ObsCandidatePaths(),
	} {
		if len(candidates) == 0 {
			t.Fatalf("%s: no candidate paths on darwin", name)
		}
		for _, c := range candidates {
			if filepath.VolumeName(c) != "" || c == "" {
				t.Fatalf("%s: unusable darwin candidate %q", name, c)
			}
		}
	}
}
