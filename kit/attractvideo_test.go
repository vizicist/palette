package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupAttractVideoDir points ConfigDir at a temp tree and returns the attract
// video directory path, without creating it. Tests that want the directory to
// exist create it themselves.
func setupAttractVideoDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	configDir := filepath.Join(root, "data_default", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PALETTE_DATAROOT", root)
	t.Setenv("PALETTE_DATA", "default")
	return filepath.Join(configDir, attractVideoDirName)
}

func writeAttractFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAttractVideoFilesMissingDirectory(t *testing.T) {
	setupAttractVideoDir(t)

	if files := attractVideoFiles(); len(files) != 0 {
		t.Fatalf("attractVideoFiles() = %v, want none when the directory does not exist", files)
	}
}

// The installer ships this directory with a README in it. That alone must not
// turn attract videos on - only actual video files do.
func TestAttractVideoFilesReadmeOnlyDoesNotCount(t *testing.T) {
	dir := setupAttractVideoDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeAttractFile(t, dir, "README.md")

	if files := attractVideoFiles(); len(files) != 0 {
		t.Fatalf("attractVideoFiles() = %v, want none when the directory holds only a README", files)
	}
}

func TestAttractVideoFilesFindsVideosInNameOrder(t *testing.T) {
	dir := setupAttractVideoDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Deliberately out of order on disk, alongside files that must be skipped.
	writeAttractFile(t, dir, "b_second.mp4")
	writeAttractFile(t, dir, "a_first.MOV") // extension match is case-insensitive
	writeAttractFile(t, dir, "c_third.mkv")
	writeAttractFile(t, dir, "README.md")
	writeAttractFile(t, dir, "notes.txt")
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	files := attractVideoFiles()
	want := []string{"a_first.MOV", "b_second.mp4", "c_third.mkv"}
	if len(files) != len(want) {
		t.Fatalf("attractVideoFiles() = %v, want %v", files, want)
	}
	for i, file := range files {
		if got := filepath.Base(file); got != want[i] {
			t.Errorf("attractVideoFiles()[%d] = %q, want %q", i, got, want[i])
		}
		if !filepath.IsAbs(file) {
			t.Errorf("attractVideoFiles()[%d] = %q, want an absolute path", i, file)
		}
	}
}

// Resolume wants file:/// URLs with forward slashes and percent-encoding, even
// on Windows, where a drive letter has to survive the conversion.
func TestResolumeFileURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a video.mp4")

	got := resolumeFileURL(path)

	if !strings.HasPrefix(got, "file:///") {
		t.Errorf("resolumeFileURL(%q) = %q, want a file:/// prefix", path, got)
	}
	if strings.Contains(got, `\`) {
		t.Errorf("resolumeFileURL(%q) = %q, want no backslashes", path, got)
	}
	if !strings.HasSuffix(got, "/a%20video.mp4") {
		t.Errorf("resolumeFileURL(%q) = %q, want the space percent-encoded", path, got)
	}
}

func TestAttractVideoAdvanceDoesNotWaitForStart(t *testing.T) {
	p := &AttractVideoPlayer{}
	// Mimic Start holding the lock during a slow Resolume REST request.
	p.mutex.Lock()
	done := make(chan struct{})
	go func() {
		p.Advance()
		close(done)
	}()

	select {
	case <-done:
		p.mutex.Unlock()
	case <-time.After(100 * time.Millisecond):
		p.mutex.Unlock()
		<-done
		t.Fatal("Advance blocked behind Start and would stall the scheduler")
	}
}
