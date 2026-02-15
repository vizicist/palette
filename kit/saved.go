package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

/*
type OldSaved struct {
	Category  string
	filename  string
	paramsmap map[string]any
}

var Saved = make(map[string]*OldSaved)

func GetSaved(name string) *OldSaved {

	p, ok := Saved[name]
	if !ok {
		category, filename := SavedNameSplit(name)
		p = &OldSaved{
			Category:  category,
			filename:  filename,
			paramsmap: make(map[string]any),
		}
		Saved[name] = p
	}
	return p
}
*/

func SavedDir() string {
	return "saved"
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
	if strings.ContainsRune(filename, os.PathSeparator) {
		return "", fmt.Errorf("SavedFilePath: filename contains path separator")
	}
	if strings.ContainsRune(filename, os.PathListSeparator) {
		return "", fmt.Errorf("SavedFilePath: filename contains path list separator")
	}
	localpath := filepath.Join(PaletteDataPath(), SavedDir(), category, filename+suffix)
	return localpath, nil
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
