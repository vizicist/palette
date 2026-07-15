package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	json "github.com/goccy/go-json"
)

// Quad presets follow a master/theme model. The base "quad" directory is the
// single master store holding the real preset JSON files (plus _Current and
// _Boot). Each pro2 theme is a "quad_<theme>" directory holding *link* files:
// small JSON pointers to a preset in the master store. A theme is therefore a
// curated subset of the master presets, with no duplicated preset data.

const quadDefaultTheme = quadCategoryBase + "_default"

// quadThemeSeededMarker records, inside the Default theme directory, that the
// theme has already been seeded. It lets the installer ship an empty Default
// directory that is seeded on first launch, while never re-seeding a Default
// theme the user has since curated. It is not a ".json" file, so it never
// appears as a preset.
const quadThemeSeededMarker = ".seeded"

// quadThemeLink is the on-disk form of a theme's link to a master quad preset.
type quadThemeLink struct {
	Link string `json:"link"`
}

// isQuadThemeCategory reports whether category is a quad *theme* directory (a
// link dir such as quad_chill or quad_default), as opposed to the master "quad"
// directory which holds the real preset files.
func isQuadThemeCategory(category string) bool {
	return IsQuadCategory(category) && category != quadCategoryBase
}

// writeQuadThemeLink creates (or overwrites) a link file in a theme directory
// pointing at the named preset in the master quad directory.
func writeQuadThemeLink(themeCategory string, presetName string) error {
	path, err := WritableSavedFilePath(themeCategory, presetName, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	data, err := json.MarshalIndent(quadThemeLink{Link: presetName}, "", "    ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// readQuadThemeLink returns the master preset name a theme link points to. A
// malformed or legacy file (e.g. a full preset copy) falls back to the link's
// own filename so it still resolves to a same-named master preset.
func readQuadThemeLink(themeCategory string, presetName string) (string, error) {
	path, err := ReadableSavedFilePath(themeCategory, presetName, ".json")
	if err != nil {
		return "", err
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var link quadThemeLink
	if err := json.Unmarshal(bytes, &link); err != nil || link.Link == "" {
		return presetName, nil
	}
	return link.Link, nil
}

// removeQuadThemeLink deletes a single link file from a theme directory.
func removeQuadThemeLink(themeCategory string, presetName string) error {
	path, err := ReadableSavedFilePath(themeCategory, presetName, ".json")
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		LogIfError(err)
		return err
	}
	return nil
}

// resolveQuadPreset maps a (category, filename) that may reference a theme link
// to the real master (category, filename) in the base quad directory. Non-theme
// categories pass through unchanged.
func resolveQuadPreset(category string, filename string) (string, string, error) {
	if !isQuadThemeCategory(category) {
		return category, filename, nil
	}
	master, err := readQuadThemeLink(category, filename)
	if err != nil {
		return "", "", err
	}
	return quadCategoryBase, master, nil
}

func quadMasterExists(name string) bool {
	path, err := ReadableSavedFilePath(quadCategoryBase, name, ".json")
	if err != nil {
		return false
	}
	return PathExists(path)
}

func quadThemeHasLink(themeCategory string, name string) bool {
	path, err := ReadableSavedFilePath(themeCategory, name, ".json")
	if err != nil {
		return false
	}
	return PathExists(path)
}

// quadThemeCategories discovers the quad theme directories that currently exist
// under the saved directory (any "quad_*" sibling of the master quad dir).
func quadThemeCategories() []string {
	base := filepath.Join(PaletteDataPath(), SavedDir())
	entries, err := os.ReadDir(base)
	if err != nil {
		LogIfError(err)
		return nil
	}
	themes := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), quadCategoryBase+"_") {
			themes = append(themes, entry.Name())
		}
	}
	return themes
}

// CopyQuadThemeLink adds a link in dstTheme to the master preset referenced by
// (srcCategory, filename). The master preset data is not duplicated.
func CopyQuadThemeLink(srcCategory string, filename string, dstTheme string) error {
	if !isQuadThemeCategory(dstTheme) {
		return fmt.Errorf("copy target %q is not a quad theme", dstTheme)
	}
	_, master, err := resolveQuadPreset(srcCategory, filename)
	if err != nil {
		return err
	}
	if !quadMasterExists(master) {
		return fmt.Errorf("preset %q not found in the quad directory", master)
	}
	if quadThemeHasLink(dstTheme, master) {
		return fmt.Errorf("preset %q is already in that theme", master)
	}
	LogInfo("CopyQuadThemeLink", "src", srcCategory, "filename", filename, "dst", dstTheme, "master", master)
	return writeQuadThemeLink(dstTheme, master)
}

// MoveQuadThemeLink relinks a preset from srcTheme to dstTheme, leaving the
// master preset untouched. It's a no-op if source and destination match.
func MoveQuadThemeLink(srcTheme string, filename string, dstTheme string) error {
	if srcTheme == dstTheme {
		return nil
	}
	if err := CopyQuadThemeLink(srcTheme, filename, dstTheme); err != nil {
		return err
	}
	if isQuadThemeCategory(srcTheme) {
		return removeQuadThemeLink(srcTheme, filename)
	}
	return nil
}

// RemoveQuadPresetOrLink removes a theme's link (curating that theme) when given
// a theme directory, or removes the master preset and every theme link to it
// when given the base quad directory.
func RemoveQuadPresetOrLink(category string, filename string) error {
	if isQuadThemeCategory(category) {
		LogInfo("RemoveQuadThemeLink", "theme", category, "filename", filename)
		return removeQuadThemeLink(category, filename)
	}
	// Base quad: remove the master preset and any theme links pointing at it.
	for _, theme := range quadThemeCategories() {
		if quadThemeHasLink(theme, filename) {
			if err := removeQuadThemeLink(theme, filename); err != nil {
				LogIfError(err)
			}
		}
	}
	return RemoveSavedFile(quadCategoryBase, filename)
}

// RenameQuadPreset renames a master quad preset and updates every theme link
// that referenced the old name. The source is always the master quad directory,
// regardless of which theme the rename was triggered from.
func RenameQuadPreset(oldName string, newName string) error {
	if err := RenameSavedFile(quadCategoryBase, oldName, newName); err != nil {
		return err
	}
	for _, theme := range quadThemeCategories() {
		if !quadThemeHasLink(theme, oldName) {
			continue
		}
		if err := writeQuadThemeLink(theme, newName); err != nil {
			LogIfError(err)
			continue
		}
		if err := removeQuadThemeLink(theme, oldName); err != nil {
			LogIfError(err)
		}
	}
	LogInfo("RenameQuadPreset", "oldName", oldName, "newName", newName)
	return nil
}

// quadThemeHasAnyLink reports whether a theme directory already holds at least
// one preset link (ignoring internal _-prefixed entries).
func quadThemeHasAnyLink(theme string) bool {
	names, err := SavedFileList(theme)
	if err != nil {
		return false
	}
	for _, name := range names {
		if !strings.HasPrefix(name, "_") {
			return true
		}
	}
	return false
}

// ensureQuadDefaultTheme seeds the Default theme the first time it's needed. The
// base quad directory is the master store, so the Default theme starts as links
// to every existing master preset. A marker file records that seeding happened,
// so a Default directory shipped empty by the installer is still seeded on first
// launch, while a Default theme the user has since curated is never re-seeded.
func ensureQuadDefaultTheme() {
	base := filepath.Join(PaletteDataPath(), SavedDir())
	defaultDir := filepath.Join(base, quadDefaultTheme)
	marker := filepath.Join(defaultDir, quadThemeSeededMarker)
	if PathExists(marker) {
		return
	}
	if err := os.MkdirAll(defaultDir, 0777); err != nil {
		LogIfError(err)
		return
	}
	// An older install may have seeded before the marker existed (or the user
	// may have already curated the theme); in that case don't seed again.
	if !quadThemeHasAnyLink(quadDefaultTheme) {
		names, err := SavedFileList(quadCategoryBase)
		if err != nil {
			LogIfError(err)
			return
		}
		seeded := 0
		for _, name := range names {
			if strings.HasPrefix(name, "_") {
				continue // skip _Current / _Boot working state
			}
			if err := writeQuadThemeLink(quadDefaultTheme, name); err != nil {
				LogIfError(err)
				continue
			}
			seeded++
		}
		LogInfo("Seeded quad_default theme", "count", seeded)
	}
	if err := os.WriteFile(marker, []byte("seeded\n"), 0644); err != nil {
		LogIfError(err)
	}
}
