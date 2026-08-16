//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vizicist/palette/internal/installerbundle"
)

// A payload that comes away cleanly reports no failures, and takes its
// directories with it.
func TestRemoveInstalledFilesReportsNoFailuresWhenClean(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bin/palette.exe", "VERSION"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	record := installRecord{
		Root:  root,
		Files: []string{"bin/palette.exe", "VERSION"},
		Dirs:  []string{"bin"},
	}
	if failed := removeInstalledFiles(record); len(failed) != 0 {
		t.Fatalf("clean removal reported failures: %v", failed)
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "palette.exe")); !os.IsNotExist(err) {
		t.Error("payload file survived a clean removal")
	}
}

// A file that cannot be removed has to be reported, not swallowed. That is what
// keeps the uninstaller and its registration in place for a retry.
func TestRemoveInstalledFilesReportsFailures(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "locked")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// A directory where the record claims a file: os.Remove fails on a
	// non-empty directory, which stands in for the locked file of the real
	// failure without needing to hold a Windows lock open.
	if err := os.WriteFile(filepath.Join(sub, "inside"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	record := installRecord{
		Root:  root,
		Files: []string{"locked"},
	}
	failed := removeInstalledFiles(record)
	if len(failed) != 1 {
		t.Fatalf("got %d failures, want 1: %v", len(failed), failed)
	}
	if !strings.Contains(failed[0], "locked") {
		t.Errorf("the failure does not name the path: %q", failed[0])
	}
}

// A file the record lists but that is already gone is not a failure - an
// interrupted uninstall that is being retried will see plenty of those.
func TestRemoveInstalledFilesIgnoresMissing(t *testing.T) {
	root := t.TempDir()
	record := installRecord{
		Root:  root,
		Files: []string{"never-existed.exe"},
	}
	if failed := removeInstalledFiles(record); len(failed) != 0 {
		t.Fatalf("an already-missing file was reported as a failure: %v", failed)
	}
}

// The shared PALETTE_DATAROOT must survive while another component still needs
// it, and an unreadable registry has to be treated as "something is still
// there" rather than deleting it.
func TestOtherPaletteComponentUninstallKeys(t *testing.T) {
	app := installerbundle.Manifest{Kind: "app"}
	data := installerbundle.Manifest{Kind: "data", DataName: "default"}

	if uninstallKey(app) == uninstallKey(data) {
		t.Fatal("the app and a data package share an uninstall key, so neither can tell the other apart")
	}
	if !strings.HasSuffix(uninstallKey(app), `\Palette`) {
		t.Errorf("unexpected app uninstall key %q", uninstallKey(app))
	}
	if !strings.HasSuffix(uninstallKey(data), `\PaletteData_default`) {
		t.Errorf("unexpected data uninstall key %q", uninstallKey(data))
	}
}
