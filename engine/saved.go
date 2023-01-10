package engine

import (
	"os"
	"path/filepath"
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

// Currently, no errors are ever returned, but log messages are generated.
func (params *ParamValues) ApplyParamsTo(category string, paramsmap map[string]any) {

	for fullname, ival := range paramsmap {
		val, okval := ival.(string)
		if !okval {
			LogWarn("value isn't a string in params json", "name", fullname, "value", val)
			continue
		}
		paramCategory, _ := SavedNameSplit(fullname)

		// Only include ones that match the category
		if (category == "layer" && IsPerLayerParam(fullname)) || category == paramCategory {
			err := params.Set(fullname, val)
			if err != nil {
				LogError(err)
				// Don't abort the whole load, i.e. we are tolerant
				// of unknown parameters or errors in the saved
			}
		}
	}
}

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

// SavedMap returns a map of saved names to file paths
// The saved names are of the form "category.name".
// If wantCategory is "*", all categories are returned
func SavedMap(wantCategory string) (map[string]string, error) {

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
		thisCategory := ""
		thisSaved := ""
		lastslash := strings.LastIndex(path, "\\")
		if lastslash >= 0 {
			thisSaved = path[lastslash+1:]
			path2 := path[0:lastslash]
			lastslash2 := strings.LastIndex(path2, "\\")
			if lastslash2 >= 0 {
				thisCategory = path2[lastslash2+1:]
			}
		}
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

// SavedArray returns a list of saved filenames for a particular category.
func SavedFileList(category string) ([]string, error) {

	savedMap, err := SavedMap(category)
	if err != nil {
		return nil, err
	}
	filelist := make([]string, 0)
	for name := range savedMap {
		name = strings.TrimPrefix(name, category+".")
		filelist = append(filelist, name)
	}
	return filelist, nil
}

func SavedList(apiargs map[string]string) (string, error) {

	wantCategory := optionalStringArg("category", apiargs, "*")
	result := "["
	sep := ""

	savedMap, err := SavedMap(wantCategory)
	if err != nil {
		return "", err
	}
	for name := range savedMap {
		thisCategory, _ := SavedNameSplit(name)
		if wantCategory == "*" || thisCategory == wantCategory {
			result += sep + "\"" + name + "\""
			sep = ","
		}
	}
	result += "]"
	return result, nil
}

// SavedFilePath returns the full path of a saved file.
func SavedFilePath(category string, filename string) string {
	jsonfile := filename + ".json"
	localpath := filepath.Join(PaletteDataPath(), SavedDir(), category, jsonfile)
	return localpath
}

// WritableSavedFilePath xxx
func WritableFilePath(category string, filename string) string {
	path := SavedFilePath(category, filename)
	LogError(os.MkdirAll(filepath.Dir(path), 0777))
	return path
}
