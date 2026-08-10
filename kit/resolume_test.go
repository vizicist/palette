package kit

import (
	"os"
	"path/filepath"
	"testing"
)

// setupSplashParamTest points ConfigDir at a temp tree and registers the splash
// image parameters, returning the config directory.
func setupSplashParamTest(t *testing.T) string {
	t.Helper()

	InitLog("") // a parameter naming a missing image logs, which needs a logger

	root := t.TempDir()
	configDir := filepath.Join(root, "data_default", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PALETTE_DATAROOT", root)
	t.Setenv("PALETTE_DATA", "default")

	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	t.Cleanup(func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
	})

	ParamDefs = map[string]ParamDef{}
	for _, paramName := range textLayerSplashImages {
		ParamDefs[paramName] = ParamDef{
			TypedParamDef: ParamDefString{},
			Category:      "global",
			Init:          "",
		}
	}
	GlobalParams = NewParamValues()

	return configDir
}

func setSplashParam(t *testing.T, name, value string) {
	t.Helper()
	if err := GlobalParams.SetParamWithString(name, value); err != nil {
		t.Fatalf("SetParamWithString(%q, %q) err = %v", name, value, err)
	}
}

func TestTextLayerSplashClipsResolvesConfiguredImages(t *testing.T) {
	configDir := setupSplashParamTest(t)
	for _, name := range []string{"sppro_startingup.png", "sppro_rebooting.png", "sppro_restarting.png"} {
		if err := os.WriteFile(filepath.Join(configDir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	setSplashParam(t, "global.resolumestartingupimage", "sppro_startingup.png")
	setSplashParam(t, "global.resolumerebootingimage", "sppro_rebooting.png")
	setSplashParam(t, "global.resolumerestartingimage", "sppro_restarting.png")

	clips := textLayerSplashClips()

	want := map[int]string{
		2: "sppro_startingup.png",
		3: "sppro_rebooting.png",
		4: "sppro_restarting.png",
	}
	if len(clips) != len(want) {
		t.Fatalf("textLayerSplashClips() = %v, want %d entries", clips, len(want))
	}
	for clipNum, wantFile := range want {
		got, ok := clips[clipNum]
		if !ok {
			t.Errorf("textLayerSplashClips() has no clip %d", clipNum)
			continue
		}
		if filepath.Base(got) != wantFile {
			t.Errorf("clip %d = %q, want %q", clipNum, got, wantFile)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("clip %d = %q, want an absolute path", clipNum, got)
		}
	}
	// Clip 1 holds the text generator and must never be rebuilt from a file.
	if path, found := clips[1]; found {
		t.Errorf("textLayerSplashClips() included clip 1 => %q, want the text generator left alone", path)
	}
}

// An installation pointed at the other image set gets that set, with no list of
// filenames in the engine to keep in step.
func TestTextLayerSplashClipsFollowsTheParameters(t *testing.T) {
	configDir := setupSplashParamTest(t)
	if err := os.WriteFile(filepath.Join(configDir, "sp_startingup.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	setSplashParam(t, "global.resolumestartingupimage", "sp_startingup.png")

	clips := textLayerSplashClips()

	if got := filepath.Base(clips[2]); got != "sp_startingup.png" {
		t.Errorf("clip 2 = %q, want sp_startingup.png", got)
	}
}

// A parameter naming a file that isn't installed is skipped, so the images that
// are present still get built.
func TestTextLayerSplashClipsSkipsMissingAndUnset(t *testing.T) {
	configDir := setupSplashParamTest(t)
	if err := os.WriteFile(filepath.Join(configDir, "sppro_rebooting.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	setSplashParam(t, "global.resolumestartingupimage", "not_installed.png")
	setSplashParam(t, "global.resolumerebootingimage", "sppro_rebooting.png")
	// global.resolumerestartingimage is left unset.

	clips := textLayerSplashClips()

	if len(clips) != 1 {
		t.Fatalf("textLayerSplashClips() = %v, want only the installed image", clips)
	}
	if got := filepath.Base(clips[3]); got != "sppro_rebooting.png" {
		t.Errorf("clip 3 = %q, want sppro_rebooting.png", got)
	}
}

// Without loaded parameters there is nothing to build, and nothing should be
// attempted - the engine must not reach Resolume in that state.
func TestTextLayerSplashClipsWithoutParams(t *testing.T) {
	old := GlobalParams
	GlobalParams = nil
	t.Cleanup(func() { GlobalParams = old })

	if clips := textLayerSplashClips(); len(clips) != 0 {
		t.Fatalf("textLayerSplashClips() = %v, want none without loaded parameters", clips)
	}
}
