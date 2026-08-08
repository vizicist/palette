package kit

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestShapeNamesFromDirAddsSVGFilesWithoutSuffix(t *testing.T) {
	dir := t.TempDir()
	for _, filename := range []string{"goat1.svg", "zebra.SVG", "chaos.svg", "notes.txt"} {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("unable to write %s: %v", filename, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "folder.svg"), 0755); err != nil {
		t.Fatalf("unable to create directory: %v", err)
	}

	got, err := shapeNamesFromDir([]string{"line", "triangle", "square", "circle", "chaos"}, dir)
	if err != nil {
		t.Fatalf("shapeNamesFromDir returned error: %v", err)
	}

	// Built-ins and SVG files are interleaved alphabetically, not listed
	// built-ins-first, so the GUI's shape list reads in order.
	want := []string{"chaos", "circle", "goat1", "line", "square", "triangle", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shapeNamesFromDir() = %#v, want %#v", got, want)
	}
}

func TestShapeNamesFromDirSortsCaseInsensitively(t *testing.T) {
	dir := t.TempDir()
	for _, filename := range []string{"Zebra.svg", "apple.svg", "Banana.svg"} {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("unable to write %s: %v", filename, err)
		}
	}

	got, err := shapeNamesFromDir([]string{"Mango", "cherry"}, dir)
	if err != nil {
		t.Fatalf("shapeNamesFromDir returned error: %v", err)
	}

	want := []string{"apple", "Banana", "cherry", "Mango", "Zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shapeNamesFromDir() = %#v, want %#v", got, want)
	}
}

// The directory may be missing; the built-ins should still come back sorted.
func TestShapeNamesFromDirSortsBaseNamesOnReadError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir")

	got, err := shapeNamesFromDir([]string{"square", "circle", "arc"}, missing)
	if err == nil {
		t.Fatal("expected an error for a missing shapes directory")
	}

	want := []string{"arc", "circle", "square"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shapeNamesFromDir() = %#v, want %#v", got, want)
	}
}
