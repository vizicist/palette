package kit

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSamplesplitterDirNamesFromDirListsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"goat", "bss", "ambient"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
			t.Fatalf("unable to create directory %s: %v", name, err)
		}
	}
	for _, filename := range []string{"loose.mp3", "README.md"} {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("unable to write %s: %v", filename, err)
		}
	}

	got, err := samplesplitterDirNamesFromDir(dir)
	if err != nil {
		t.Fatalf("samplesplitterDirNamesFromDir returned error: %v", err)
	}

	want := []string{"", "ambient", "bss", "goat"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("samplesplitterDirNamesFromDir() = %#v, want %#v", got, want)
	}
}

func TestSamplesplitterDirNamesFromDirKeepsEmptyChoiceOnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir")

	got, err := samplesplitterDirNamesFromDir(missing)
	if err == nil {
		t.Fatal("expected an error for a missing directory")
	}

	want := []string{""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("samplesplitterDirNamesFromDir() = %#v, want %#v", got, want)
	}
}
