package samplesplitter

import (
	"errors"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestListAndChoosePrefixedMP3(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"oracle-2.mp3",
		"chaos-1.mp3",
		"notes.txt",
		"sacred-1.MP3",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := ListMP3Files(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(files), 3; got != want {
		t.Fatalf("len(files) = %d, want %d", got, want)
	}
	if files[0].Name != "chaos-1.mp3" {
		t.Fatalf("files sorted with chaos first, got %q", files[0].Name)
	}

	chosen, err := ChooseRandomPrefixedMP3(dir, "sacred", rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Name != "sacred-1.MP3" {
		t.Fatalf("chosen.Name = %q", chosen.Name)
	}

	_, err = ChooseRandomPrefixedMP3(dir, "directive", rand.New(rand.NewSource(1)))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing prefix error = %v, want fs.ErrNotExist", err)
	}
}

func TestResolveMP3FileRequiresDirectChildMP3(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "sample.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveMP3File(dir, "sample.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(resolved) != "sample.mp3" {
		t.Fatalf("resolved = %q", resolved)
	}

	for _, name := range []string{"nested/sample.mp3", "../sample.mp3", "sample.wav"} {
		if _, err := ResolveMP3File(dir, name); err == nil {
			t.Fatalf("ResolveMP3File(%q) succeeded, want error", name)
		}
	}
}
