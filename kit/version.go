package kit

// The Palette version is compiled into every binary, so a binary can say what
// it is even when it isn't running out of an installed tree.  The copy here is
// the one that gets embedded; the VERSION file at the source root stays put
// because the build scripts read it (batch and shell can't read a Go constant)
// and because isPaletteRoot uses its presence to recognize a Palette root.
// TestVersionMatchesSourceRoot keeps the two from drifting apart.

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed version.txt
var paletteVersion string

// GetPaletteVersion returns the version this binary was built from.
func GetPaletteVersion() string {
	return strings.TrimSpace(paletteVersion)
}

// InstalledVersion returns the version in the VERSION file of the Palette root
// this binary is running out of, or "" if it can't be read.  This can differ
// from GetPaletteVersion, since the binaries and the data are separate
// installers and can be installed from different releases.
func InstalledVersion() string {
	b, err := os.ReadFile(filepath.Join(PaletteDir(), "VERSION"))
	if err != nil {
		return "" // It's okay if the file isn't present
	}
	return strings.TrimSpace(string(b))
}

// LogVersionMismatch warns when the installed files came from a different
// release than this binary.
func LogVersionMismatch() {
	installed := InstalledVersion()
	if installed != "" && installed != GetPaletteVersion() {
		LogWarn("Version mismatch between this binary and the installed files",
			"binary", GetPaletteVersion(), "installed", installed)
	}
}
