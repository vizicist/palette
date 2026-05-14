package samplesplitter

import (
	"errors"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type MP3File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func ListMP3Files(dir string) ([]MP3File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]MP3File, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".mp3" {
			continue
		}
		files = append(files, MP3File{
			Name: entry.Name(),
			Path: filepath.Join(dir, entry.Name()),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	return files, nil
}

func ChooseRandomPrefixedMP3(dir, prefix string, rng *rand.Rand) (MP3File, error) {
	files, err := ListMP3Files(dir)
	if err != nil {
		return MP3File{}, err
	}
	matches := make([]MP3File, 0)
	for _, file := range files {
		if strings.HasPrefix(strings.ToLower(file.Name), strings.ToLower(prefix)) {
			matches = append(matches, file)
		}
	}
	if len(matches) == 0 {
		return MP3File{}, fs.ErrNotExist
	}
	if rng == nil {
		return matches[rand.Intn(len(matches))], nil
	}
	return matches[rng.Intn(len(matches))], nil
}

func ResolveMP3File(dir, filename string) (string, error) {
	if filename == "" {
		return "", errors.New("missing file")
	}

	base, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(base, filename))
	if err != nil {
		return "", err
	}
	if filepath.Dir(candidate) != base || strings.ToLower(filepath.Ext(candidate)) != ".mp3" {
		return "", errors.New("file must be an MP3 directly inside the configured directory")
	}
	if _, err := os.Stat(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}
