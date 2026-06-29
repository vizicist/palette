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

	want := []string{"line", "triangle", "square", "circle", "chaos", "goat1", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shapeNamesFromDir() = %#v, want %#v", got, want)
	}
}
