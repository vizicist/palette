package kit

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	json "github.com/goccy/go-json"
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

// setupAttractVideoDestination points global.attractvideodestination at one
// value for the duration of a test, restoring the global parameter state after.
func setupAttractVideoDestination(t *testing.T, dest string) {
	t.Helper()

	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	t.Cleanup(func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
	})

	ParamDefs = map[string]ParamDef{
		"global.attractvideodestination": {
			Category:      "global",
			Init:          attractVideoDestMain,
			TypedParamDef: ParamDefString{},
		},
	}
	GlobalParams = NewParamValues()
	if err := GlobalParams.SetParamWithString("global.attractvideodestination", dest); err != nil {
		t.Fatalf("set global.attractvideodestination: %v", err)
	}
}

func TestNormalizeAttractVideoDestination(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"main", attractVideoDestMain},
		{"gui", attractVideoDestGUI},
		{"GUI", attractVideoDestGUI},
		{"  gui  ", attractVideoDestGUI},
		{"", attractVideoDestMain},
		// A typo must leave the videos where they have always been, not make
		// them vanish from both destinations.
		{"guy", attractVideoDestMain},
		{"resolume", attractVideoDestMain},
	} {
		if got := normalizeAttractVideoDestination(tc.in); got != tc.want {
			t.Errorf("normalizeAttractVideoDestination(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAttractVideoDestinationReadsTheParameter(t *testing.T) {
	setupAttractVideoDestination(t, attractVideoDestGUI)

	if got := AttractVideoDestination(); got != attractVideoDestGUI {
		t.Errorf("AttractVideoDestination() = %q, want %q", got, attractVideoDestGUI)
	}
}

// Resolume asks where the videos go while it is starting up, which can be
// before the parameters are loaded.
func TestAttractVideoDestinationWithoutParams(t *testing.T) {
	oldGlobalParams := GlobalParams
	GlobalParams = nil
	t.Cleanup(func() { GlobalParams = oldGlobalParams })

	if got := AttractVideoDestination(); got != attractVideoDestMain {
		t.Errorf("AttractVideoDestination() = %q, want %q", got, attractVideoDestMain)
	}
}

// The files/loaded bookkeeping records what has been pushed into Resolume as
// clips. A run on the GUI screen pushes nothing, so it must not touch it: if it
// recorded its own file list there, a later switch back to the Resolume
// destination would find the list unchanged, believe those files were already
// loaded, and skip the load - leaving it connecting clips that were never
// opened.
func TestAttractVideoStartGUILeavesResolumeClipsAlone(t *testing.T) {
	loadedFiles := []string{"one.mp4", "two.mp4"}
	p := &AttractVideoPlayer{
		files:     loadedFiles,
		durations: []float64{12, 34},
		loaded:    true,
	}

	p.startGUI([]string{"one.mp4", "two.mp4", "three.mp4"})

	if !slices.Equal(p.files, loadedFiles) {
		t.Errorf("startGUI changed the loaded clip list to %v, want %v", p.files, loadedFiles)
	}
	if !p.loaded {
		t.Error("startGUI cleared loaded, so a later Resolume run would not reload the clips it still has")
	}
	if !p.playing {
		t.Error("startGUI did not mark the player playing, so Stop would do nothing")
	}
	if p.dest != attractVideoDestGUI {
		t.Errorf("startGUI recorded dest %q, want %q", p.dest, attractVideoDestGUI)
	}
}

// The browser moves to the next file when the one it is playing ends, so the
// tick must leave the playlist alone - both because it has nothing to send and
// because advancing here would put the engine's idea of the current file out of
// step with the browser's.
func TestAttractVideoAdvanceGUILeavesThePlaylistAlone(t *testing.T) {
	p := &AttractVideoPlayer{
		files:      []string{"one.mp4", "two.mp4", "three.mp4"},
		durations:  []float64{1, 1, 1},
		playing:    true,
		dest:       attractVideoDestGUI,
		nextSwitch: time.Now().Add(-time.Minute), // long overdue
	}

	p.Advance()

	if p.current != 0 {
		t.Errorf("Advance moved the GUI playlist to %d, want it left at 0", p.current)
	}
}

func TestAttractVideoStopGUIEndsThePlay(t *testing.T) {
	p := &AttractVideoPlayer{playing: true, dest: attractVideoDestGUI}

	p.Stop()

	if p.playing {
		t.Error("Stop left the player marked playing")
	}
	// Stopping twice must stay harmless - the second call is what a Restart
	// after attract mode has already ended looks like.
	p.Stop()
}

// The browser has one thing to test - an empty file list - so the list must
// marshal as [] rather than null when there is nothing to play.
func TestAttractVideoListJSONWithNothingToPlay(t *testing.T) {
	setupAttractVideoDir(t)
	setupAttractVideoDestination(t, attractVideoDestGUI)

	got, err := AttractVideoListJSON()
	if err != nil {
		t.Fatalf("AttractVideoListJSON: %v", err)
	}

	var playlist AttractVideoPlaylist
	if err := json.Unmarshal([]byte(got), &playlist); err != nil {
		t.Fatalf("unmarshal %q: %v", got, err)
	}
	if playlist.Destination != attractVideoDestGUI {
		t.Errorf("destination = %q, want %q", playlist.Destination, attractVideoDestGUI)
	}
	if playlist.Files == nil {
		t.Errorf("files = null in %q, want an empty array", got)
	}
	if len(playlist.Files) != 0 {
		t.Errorf("files = %v, want none", playlist.Files)
	}
}
