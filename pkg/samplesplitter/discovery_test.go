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
		"sacred-1.MP3",
	} {
		if err := writeTestMP3(filepath.Join(dir, name), 11); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeTestMP3(filepath.Join(dir, "short.mp3"), 9); err != nil {
		t.Fatal(err)
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

func TestChooseRandomPrefixedMP3ExcludingAvoidsRepeatWhenPossible(t *testing.T) {
	dir := t.TempDir()
	sacred1 := filepath.Join(dir, "sacred-1.mp3")
	sacred2 := filepath.Join(dir, "sacred-2.mp3")
	for _, path := range []string{sacred1, sacred2} {
		if err := writeTestMP3(path, 11); err != nil {
			t.Fatal(err)
		}
	}

	chosen, err := ChooseRandomPrefixedMP3Excluding(dir, "sacred", sacred1, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Path == sacred1 {
		t.Fatalf("chosen.Path = %q, want alternate from excluded previous sample", chosen.Path)
	}
	if chosen.Path != sacred2 {
		t.Fatalf("chosen.Path = %q, want %q", chosen.Path, sacred2)
	}
}

func TestChooseRandomPrefixedMP3ExcludingFallsBackToOnlyMatch(t *testing.T) {
	dir := t.TempDir()
	only := filepath.Join(dir, "oracle-1.mp3")
	if err := writeTestMP3(only, 11); err != nil {
		t.Fatal(err)
	}

	chosen, err := ChooseRandomPrefixedMP3Excluding(dir, "oracle", only, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Path != only {
		t.Fatalf("chosen.Path = %q, want only matching file %q", chosen.Path, only)
	}
}

func TestResolveMP3FileRequiresDirectChildMP3(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestMP3(filepath.Join(dir, "sample.mp3"), 11); err != nil {
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

func TestResolveMP3FileRejectsShortMP3(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestMP3(filepath.Join(dir, "short.mp3"), 9); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveMP3File(dir, "short.mp3"); err == nil {
		t.Fatal("ResolveMP3File(short.mp3) succeeded, want short duration error")
	}
}

func TestMinimumMP3DurationCanBeLowered(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestMP3(filepath.Join(dir, "short.mp3"), 2); err != nil {
		t.Fatal(err)
	}

	files, err := ListMP3FilesWithMinimumDuration(dir, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(files), 1; got != want {
		t.Fatalf("len(files) = %d, want %d", got, want)
	}

	if _, err := ResolveMP3FileWithMinimumDuration(dir, "short.mp3", 1.0); err != nil {
		t.Fatalf("ResolveMP3FileWithMinimumDuration err = %v", err)
	}
}

func writeTestMP3(path string, seconds int) error {
	header := []byte{0xff, 0xfb, 0x90, 0x64}
	frameLen := 417
	frame := make([]byte, frameLen)
	copy(frame, header)
	frameCount := seconds * 39
	data := make([]byte, 0, frameLen*frameCount)
	for i := 0; i < frameCount; i++ {
		data = append(data, frame...)
	}
	return os.WriteFile(path, data, 0o644)
}
