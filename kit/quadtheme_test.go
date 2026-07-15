package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func setupQuadThemeTest(t *testing.T) {
	t.Helper()
	t.Setenv("PALETTE_DATAROOT", t.TempDir())
	InitLog("")
}

func TestResolveQuadPreset(t *testing.T) {
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	if err := writeQuadThemeLink("quad_chill", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink: %v", err)
	}

	cat, name, err := resolveQuadPreset("quad_chill", "Groove")
	if err != nil {
		t.Fatalf("resolveQuadPreset: %v", err)
	}
	if cat != "quad" || name != "Groove" {
		t.Fatalf("resolveQuadPreset = (%q,%q), want (quad,Groove)", cat, name)
	}

	// The master quad directory passes through unchanged.
	cat, name, err = resolveQuadPreset("quad", "Groove")
	if err != nil {
		t.Fatalf("resolveQuadPreset master: %v", err)
	}
	if cat != "quad" || name != "Groove" {
		t.Fatalf("resolveQuadPreset master = (%q,%q), want (quad,Groove)", cat, name)
	}
}

func TestCopyQuadThemeLink(t *testing.T) {
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	if err := writeQuadThemeLink("quad_chill", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink: %v", err)
	}

	if err := CopyQuadThemeLink("quad_chill", "Groove", "quad_melodic"); err != nil {
		t.Fatalf("CopyQuadThemeLink: %v", err)
	}
	if !quadThemeHasLink("quad_melodic", "Groove") {
		t.Fatal("preset should be linked in the destination theme")
	}
	if !quadThemeHasLink("quad_chill", "Groove") {
		t.Fatal("source link should remain after copy")
	}
	if !savedPresetExists(t, "quad", "Groove") {
		t.Fatal("master preset should remain after copy")
	}

	// Copying onto an existing theme link fails without clobbering.
	if err := CopyQuadThemeLink("quad_chill", "Groove", "quad_melodic"); err == nil {
		t.Fatal("copy onto existing theme link should fail")
	}
	// Copying a preset with no master file fails.
	if err := CopyQuadThemeLink("quad_chill", "Nope", "quad_melodic"); err == nil {
		t.Fatal("copy of a missing preset should fail")
	}
}

func TestCopyFromMasterAllView(t *testing.T) {
	// The "All" theme is backed by the master quad directory. Copying an
	// orphan preset (present in the master, linked into no theme) from the All
	// view into a curated theme must add a link there.
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Orphan")

	if err := CopyQuadThemeLink("quad", "Orphan", "quad_chill"); err != nil {
		t.Fatalf("CopyQuadThemeLink from master: %v", err)
	}
	if !quadThemeHasLink("quad_chill", "Orphan") {
		t.Fatal("orphan should be linked into the target theme")
	}
	if !savedPresetExists(t, "quad", "Orphan") {
		t.Fatal("master preset should remain after copy from All")
	}
}

func TestMoveQuadThemeLink(t *testing.T) {
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	if err := writeQuadThemeLink("quad_chill", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink: %v", err)
	}

	if err := MoveQuadThemeLink("quad_chill", "Groove", "quad_melodic"); err != nil {
		t.Fatalf("MoveQuadThemeLink: %v", err)
	}
	if quadThemeHasLink("quad_chill", "Groove") {
		t.Fatal("source link should be gone after move")
	}
	if !quadThemeHasLink("quad_melodic", "Groove") {
		t.Fatal("destination link should exist after move")
	}
	if !savedPresetExists(t, "quad", "Groove") {
		t.Fatal("master preset should remain after move")
	}
}

func TestRemoveQuadThemeLinkCuratesOnly(t *testing.T) {
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	if err := writeQuadThemeLink("quad_chill", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink chill: %v", err)
	}
	if err := writeQuadThemeLink("quad_melodic", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink melodic: %v", err)
	}

	if err := RemoveQuadPresetOrLink("quad_chill", "Groove"); err != nil {
		t.Fatalf("RemoveQuadPresetOrLink: %v", err)
	}
	if quadThemeHasLink("quad_chill", "Groove") {
		t.Fatal("chill link should be removed")
	}
	if !quadThemeHasLink("quad_melodic", "Groove") {
		t.Fatal("other theme link should remain")
	}
	if !savedPresetExists(t, "quad", "Groove") {
		t.Fatal("master preset should remain after removing a theme link")
	}
}

func TestRemoveQuadMasterRemovesAllLinks(t *testing.T) {
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	if err := writeQuadThemeLink("quad_chill", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink chill: %v", err)
	}
	if err := writeQuadThemeLink("quad_melodic", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink melodic: %v", err)
	}

	if err := RemoveQuadPresetOrLink("quad", "Groove"); err != nil {
		t.Fatalf("RemoveQuadPresetOrLink master: %v", err)
	}
	if savedPresetExists(t, "quad", "Groove") {
		t.Fatal("master preset should be gone")
	}
	if quadThemeHasLink("quad_chill", "Groove") || quadThemeHasLink("quad_melodic", "Groove") {
		t.Fatal("all theme links should be gone with the master")
	}
}

func TestRenameQuadPresetUpdatesLinks(t *testing.T) {
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	if err := writeQuadThemeLink("quad_chill", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink chill: %v", err)
	}
	if err := writeQuadThemeLink("quad_melodic", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink melodic: %v", err)
	}

	if err := RenameQuadPreset("Groove", "Smooth"); err != nil {
		t.Fatalf("RenameQuadPreset: %v", err)
	}
	if savedPresetExists(t, "quad", "Groove") {
		t.Fatal("old master should be gone")
	}
	if !savedPresetExists(t, "quad", "Smooth") {
		t.Fatal("new master should exist")
	}
	for _, theme := range []string{"quad_chill", "quad_melodic"} {
		if quadThemeHasLink(theme, "Groove") {
			t.Fatalf("%s: old link should be gone", theme)
		}
		if !quadThemeHasLink(theme, "Smooth") {
			t.Fatalf("%s: new link should exist", theme)
		}
		master, err := readQuadThemeLink(theme, "Smooth")
		if err != nil || master != "Smooth" {
			t.Fatalf("%s: link should point to Smooth, got %q err %v", theme, master, err)
		}
	}
}

func TestQuadSaveWritesMasterAndThemeLink(t *testing.T) {
	setupQuadThemeTest(t)

	quad := &Quad{patch: map[string]*Patch{}}
	if err := quad.save("quad_chill", "Groove"); err != nil {
		t.Fatalf("quad.save: %v", err)
	}

	// The real preset must land in the master quad directory.
	if !savedPresetExists(t, "quad", "Groove") {
		t.Fatal("save should write the preset to the master quad directory")
	}
	// The current theme must get a link (not a full copy) to it.
	if !quadThemeHasLink("quad_chill", "Groove") {
		t.Fatal("save should write a link into the current theme")
	}
	master, err := readQuadThemeLink("quad_chill", "Groove")
	if err != nil {
		t.Fatalf("readQuadThemeLink: %v", err)
	}
	if master != "Groove" {
		t.Fatalf("theme entry should be a link to Groove, got %q", master)
	}
}

func TestEnsureQuadDefaultThemeSeedsLinks(t *testing.T) {
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	writeTestPreset(t, "quad", "Mellow")
	writeTestPreset(t, "quad", "_Current") // working state, must be skipped

	ensureQuadDefaultTheme()

	if !quadThemeHasLink("quad_default", "Groove") {
		t.Fatal("Groove should be seeded into Default")
	}
	if !quadThemeHasLink("quad_default", "Mellow") {
		t.Fatal("Mellow should be seeded into Default")
	}
	if quadThemeHasLink("quad_default", "_Current") {
		t.Fatal("_Current should not be seeded")
	}

	// After seeding, the marker means later curation sticks: removing a link
	// and re-running must not re-add it.
	if err := removeQuadThemeLink("quad_default", "Groove"); err != nil {
		t.Fatalf("removeQuadThemeLink: %v", err)
	}
	ensureQuadDefaultTheme()
	if quadThemeHasLink("quad_default", "Groove") {
		t.Fatal("ensureQuadDefaultTheme should not re-seed after the marker is written")
	}
}

func TestEnsureQuadDefaultThemeSeedsPreCreatedEmptyDir(t *testing.T) {
	// The installer ships an empty Default directory (a placeholder file only);
	// first launch must still seed it.
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	writeTestPreset(t, "quad", "Mellow")

	base := filepath.Join(PaletteDataPath(), SavedDir())
	dir := filepath.Join(base, "quad_default")
	if err := os.MkdirAll(dir, 0777); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("placeholder\n"), 0644); err != nil {
		t.Fatalf("write placeholder: %v", err)
	}

	ensureQuadDefaultTheme()

	if !quadThemeHasLink("quad_default", "Groove") || !quadThemeHasLink("quad_default", "Mellow") {
		t.Fatal("an installer-created empty Default directory should still be seeded")
	}
}

func TestEnsureQuadDefaultThemeDoesNotReseedCuratedDir(t *testing.T) {
	// Simulate an older install that seeded before markers existed and where the
	// user curated Default down to a single preset. Seeding must not re-add the
	// dropped presets.
	setupQuadThemeTest(t)
	writeTestPreset(t, "quad", "Groove")
	writeTestPreset(t, "quad", "Mellow")
	if err := writeQuadThemeLink("quad_default", "Groove"); err != nil {
		t.Fatalf("writeQuadThemeLink: %v", err)
	}

	ensureQuadDefaultTheme()

	if quadThemeHasLink("quad_default", "Mellow") {
		t.Fatal("an already-curated Default must not be re-seeded")
	}
	if !quadThemeHasLink("quad_default", "Groove") {
		t.Fatal("the existing curated link should remain")
	}
}
