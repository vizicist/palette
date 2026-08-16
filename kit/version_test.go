package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Bumping a release means editing both kit/version.txt and the VERSION file at
// the source root: the binaries report the embedded copy, while the build
// scripts and isPaletteRoot use the file.  This catches forgetting one.
func TestVersionMatchesSourceRoot(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "VERSION"))
	if err != nil {
		t.Fatalf("reading source root VERSION: %v", err)
	}
	root := strings.TrimSpace(string(b))
	if got := GetPaletteVersion(); got != root {
		t.Errorf("version drift: kit/version.txt is %q, VERSION is %q (bump both)", got, root)
	}
}

func TestGetPaletteVersionIsUsable(t *testing.T) {
	v := GetPaletteVersion()
	if v == "" {
		t.Fatal("GetPaletteVersion is empty, version.txt didn't get embedded")
	}
	// The version goes into release filenames (palette_8.39_win_setup.exe),
	// so it has to survive as a single bare token.
	if strings.ContainsAny(v, " \t\r\n") {
		t.Errorf("GetPaletteVersion has whitespace in it: %q", v)
	}
}
