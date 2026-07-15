package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

func SavedDir() string {
	return "saved"
}

// quadCategoryBase is the saved category/directory for the Default theme's
// quad presets. Other pro2 themes use "quad_<theme>" sibling directories.
const quadCategoryBase = "quad"

// IsQuadCategory reports whether a saved category holds quad-format presets.
// The base "quad" directory is the Default theme; each additional pro2 theme
// uses a "quad_<theme>" directory (e.g. quad_chill). All of them store the same
// quad-format files, so they share the quad load/save code paths.
func IsQuadCategory(category string) bool {
	return category == quadCategoryBase || strings.HasPrefix(category, quadCategoryBase+"_")
}

func SavedNameSplit(saved string) (string, string) {
	words := strings.SplitN(saved, ".", 2)
	if len(words) == 1 {
		return "", words[0]
	} else {
		return words[0], words[1]
	}
}

// SavedPathMap returns a map of saved names to file paths
// The saved names are of the form "category.name".
// If wantCategory is "*", all categories are returned
func SavedPathMap(wantCategory string) (map[string]string, error) {
	if wantCategory != "*" {
		if err := validateSavedPathPart("category", wantCategory); err != nil {
			return nil, err
		}
	}

	result := make(map[string]string, 0)

	walker := func(walkedpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Only look at .json files
		if !strings.HasSuffix(walkedpath, ".json") {
			return nil
		}
		path := strings.TrimSuffix(walkedpath, ".json")
		// the last two components of the path are category and saved
		thisSaved := filepath.Base(path)
		catdir := filepath.Dir(path)
		thisCategory := filepath.Base(catdir)
		if wantCategory == "*" || thisCategory == wantCategory {
			result[thisCategory+"."+thisSaved] = walkedpath
		}
		return nil
	}

	savedDir1 := filepath.Join(PaletteDataPath(), SavedDir())
	err := filepath.Walk(savedDir1, walker)
	if err != nil {
		LogWarn("filepath.Walk", "err", err)
		return nil, err
	}
	return result, nil
}

// SavedFileList returns a list of saved filenames for a particular category.
func SavedFileList(category string) ([]string, error) {

	savedMap, err := SavedPathMap(category)
	if err != nil {
		return nil, err
	}
	filelist := make([]string, 0)
	for name := range savedMap {
		name = strings.TrimPrefix(name, category+".")
		filelist = append(filelist, name)
	}
	slices.Sort(filelist)
	return filelist, nil
}

func SavedListAsString(category string) (string, error) {

	result := "["
	sep := ""

	savedMap, err := SavedPathMap(category)
	if err != nil {
		return "", err
	}
	for name := range savedMap {
		thisCategory, _ := SavedNameSplit(name)
		if category == "*" || thisCategory == category {
			result += sep + "\"" + name + "\""
			sep = ","
		}
	}
	result += "]"
	return result, nil
}

// ReadableSavedFilePath returns the full path of a saved file.
// The value of filename isn't trusted, verify its sanity.
func ReadableSavedFilePath(category string, filename string, suffix string) (string, error) {
	if err := validateSavedPathPart("category", category); err != nil {
		return "", err
	}
	if err := validateSavedPathPart("filename", filename); err != nil {
		return "", err
	}
	if suffix != ".json" && suffix != ".zip" {
		return "", fmt.Errorf("SavedFilePath: unsupported suffix %q", suffix)
	}

	localpath := filepath.Join(PaletteDataPath(), SavedDir(), category, filename+suffix)
	if err := validateSavedPathWithinBase(localpath); err != nil {
		return "", err
	}
	return localpath, nil
}

func validateSavedPathWithinBase(path string) error {
	base := filepath.Join(PaletteDataPath(), SavedDir())
	absBase, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("SavedFilePath: unable to resolve saved base: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("SavedFilePath: unable to resolve saved path: %w", err)
	}
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return fmt.Errorf("SavedFilePath: unable to compare saved path: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("SavedFilePath: path escapes saved directory")
	}
	return nil
}

func validateSavedPathPart(kind string, value string) error {
	if value == "" {
		return fmt.Errorf("SavedFilePath: %s is empty", kind)
	}
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("SavedFilePath: %s has leading or trailing whitespace", kind)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("SavedFilePath: %s cannot be %q", kind, value)
	}
	if len(value) > 120 {
		return fmt.Errorf("SavedFilePath: %s is too long", kind)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("SavedFilePath: %s is not valid UTF-8", kind)
	}
	if strings.ContainsAny(value, `/\:*?"<>|`) {
		return fmt.Errorf("SavedFilePath: %s contains a reserved character", kind)
	}
	if strings.HasSuffix(value, ".") || strings.HasSuffix(value, " ") {
		return fmt.Errorf("SavedFilePath: %s cannot end with a dot or space", kind)
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("SavedFilePath: %s contains a control character", kind)
		}
	}
	if isWindowsReservedFilename(value) {
		return fmt.Errorf("SavedFilePath: %s uses a reserved Windows name", kind)
	}
	return nil
}

func isWindowsReservedFilename(value string) bool {
	base := strings.ToUpper(value)
	if dot := strings.IndexByte(base, '.'); dot >= 0 {
		base = base[:dot]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	for _, prefix := range []string{"COM", "LPT"} {
		if strings.HasPrefix(base, prefix) && len(base) == len(prefix)+1 {
			n := base[len(prefix)]
			if n >= '1' && n <= '9' {
				return true
			}
		}
	}
	return false
}

// WritableSavedFilePath xxx
func WritableSavedFilePath(category string, filename string, suffix string) (string, error) {
	path, err := ReadableSavedFilePath(category, filename, suffix)
	if err != nil {
		LogIfError(err)
		return "", err
	}
	// Create all the directories we need
	// in order for this file to be writable.
	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		LogIfError(err)
		return "", err
	}
	return path, nil
}

// RenameSavedFile renames a saved preset within the same category directory.
// It fails if a preset with the new name already exists so nothing is
// clobbered.
func RenameSavedFile(category string, oldname string, newname string) error {
	src, err := ReadableSavedFilePath(category, oldname, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	dst, err := WritableSavedFilePath(category, newname, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	if src == dst {
		return nil
	}
	if PathExists(dst) {
		return fmt.Errorf("a preset named %q already exists", newname)
	}
	if err := os.Rename(src, dst); err != nil {
		LogIfError(err)
		return err
	}
	LogInfo("RenameSavedFile", "category", category, "oldname", oldname, "newname", newname)
	return nil
}

// MoveSavedFile moves a saved preset from one category directory to another,
// keeping the same filename. It fails if the destination already exists so
// nothing is clobbered. This is used to move quad presets between pro2 themes.
func MoveSavedFile(srcCategory string, filename string, dstCategory string) error {
	src, err := ReadableSavedFilePath(srcCategory, filename, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	dst, err := WritableSavedFilePath(dstCategory, filename, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	if src == dst {
		return nil
	}
	if PathExists(dst) {
		return fmt.Errorf("a preset named %q already exists in %q", filename, dstCategory)
	}
	if err := os.Rename(src, dst); err != nil {
		LogIfError(err)
		return err
	}
	LogInfo("MoveSavedFile", "srcCategory", srcCategory, "filename", filename, "dstCategory", dstCategory)
	return nil
}

// CopySavedFile copies a saved preset from one category directory to another,
// keeping the same filename and leaving the original in place. It fails if the
// destination already exists so nothing is clobbered. This is used to copy quad
// presets between pro2 themes.
func CopySavedFile(srcCategory string, filename string, dstCategory string) error {
	src, err := ReadableSavedFilePath(srcCategory, filename, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	dst, err := WritableSavedFilePath(dstCategory, filename, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	if src == dst {
		return fmt.Errorf("cannot copy a preset onto itself")
	}
	if PathExists(dst) {
		return fmt.Errorf("a preset named %q already exists in %q", filename, dstCategory)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		LogIfError(err)
		return err
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		LogIfError(err)
		return err
	}
	LogInfo("CopySavedFile", "srcCategory", srcCategory, "filename", filename, "dstCategory", dstCategory)
	return nil
}

func RemoveSavedFile(category string, filename string) error {
	path, err := ReadableSavedFilePath(category, filename, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	err = os.Remove(path)
	if err != nil {
		LogIfError(err)
		return err
	}
	LogInfo("RemoveSavedFile", "category", category, "filename", filename, "path", path)
	return nil
}
