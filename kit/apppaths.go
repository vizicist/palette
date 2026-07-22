package kit

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Locating third-party applications whose install locations differ per OS.
// The configured global.*path parameter always wins; the candidates here are
// the fallback used when that parameter is unset or points at nothing, which
// is the normal case on macOS since the shipped defaults are Windows paths.

// firstExistingPath returns the first candidate that exists. A candidate
// containing glob metacharacters is expanded, and its matches are tried in
// reverse order so a versioned app bundle (Bidule 0.9776 P64.app) resolves to
// the highest version installed.
func firstExistingPath(candidates []string) string {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if !strings.ContainsAny(candidate, "*?[") {
			if FileExists(candidate) {
				return candidate
			}
			continue
		}
		matches, err := filepath.Glob(candidate)
		if err != nil {
			LogIfError(err)
			continue
		}
		sort.Sort(sort.Reverse(sort.StringSlice(matches)))
		for _, match := range matches {
			if FileExists(match) {
				return match
			}
		}
	}
	return ""
}

// homeJoin builds a path under the user's home directory, or "" if $HOME is unset.
func homeJoin(elem ...string) string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(append([]string{home}, elem...)...)
}

// lookPath finds an executable on $PATH, returning "" if it isn't there.
func lookPath(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// BiduleCandidatePaths lists where Plogue Bidule is normally installed.
// The macOS bundle name carries the version and edition (e.g.
// "Bidule 0.9776 P64.app"), so those entries are globs.
func BiduleCandidatePaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Bidule*.app/Contents/MacOS/Bidule*",
			"/Applications/Plogue/Bidule*.app/Contents/MacOS/Bidule*",
			homeJoin("Applications", "Bidule*.app", "Contents", "MacOS", "Bidule*"),
		}
	case "windows":
		return []string{
			"C:/Program Files/Plogue/Bidule/Bidule.exe",
			"C:/Program Files (x86)/Plogue/Bidule/Bidule.exe",
		}
	default:
		return []string{
			lookPath("bidule"),
			"/usr/local/bin/bidule",
		}
	}
}

// ObsCandidatePaths lists where OBS Studio is normally installed.
func ObsCandidatePaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/OBS.app/Contents/MacOS/OBS",
			homeJoin("Applications", "OBS.app", "Contents", "MacOS", "OBS"),
		}
	case "windows":
		return []string{
			"C:/Program Files/obs-studio/bin/64bit/obs64.exe",
			"C:/Program Files (x86)/obs-studio/bin/64bit/obs64.exe",
		}
	default:
		return []string{
			lookPath("obs"),
			"/usr/bin/obs",
		}
	}
}

// ObsConfigDir is where OBS Studio keeps its per-user configuration, which is
// what holds the .sentinel folder deleted before launch.
func ObsConfigDir() string {
	switch runtime.GOOS {
	case "darwin":
		return homeJoin("Library", "Application Support", "obs-studio")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return ""
		}
		return filepath.Join(appData, "obs-studio")
	default:
		if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
			return filepath.Join(configHome, "obs-studio")
		}
		return homeJoin(".config", "obs-studio")
	}
}

// appExeName is the process name to kill or look for. On macOS the executable
// inside the app bundle is the process name, and it isn't predictable (Bidule
// names its binary "Bidule P64"), so the installed bundle is consulted first.
func appExeName(base string, darwinDefault string, candidates []string) string {
	if runtime.GOOS != "darwin" {
		return executableName(base)
	}
	if path := firstExistingPath(candidates); path != "" {
		return filepath.Base(path)
	}
	return darwinDefault
}
