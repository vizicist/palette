package installerbundle

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPackAndRead(t *testing.T) {
	tmp := t.TempDir()
	stub := filepath.Join(tmp, "stub.exe")
	source := filepath.Join(tmp, "source")
	output := filepath.Join(tmp, "installer.exe")
	if err := os.WriteFile(stub, []byte("native-stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "bin", "palette.exe"), []byte("payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	wantManifest := Manifest{Kind: "app", Version: "1.2.3", Delete: []string{"old.exe"}}
	if err := Pack(stub, source, output, wantManifest); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, _ := f.Stat()
	bundle, err := Read(f, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	if bundle.StubSize != int64(len("native-stub")) || bundle.Manifest.Version != "1.2.3" {
		t.Fatalf("unexpected bundle metadata: %#v", bundle)
	}
	if len(bundle.Archive.File) != 2 || bundle.Archive.File[1].Name != "bin/palette.exe" {
		t.Fatalf("unexpected archive: %#v", bundle.Archive.File)
	}
	r, err := bundle.Archive.File[1].Open()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	r.Close()
	if string(got) != "payload" {
		t.Fatalf("payload = %q", got)
	}
}

func TestIsPresetPath(t *testing.T) {
	for _, name := range []string{"saved", "saved/quad/Foo.json", "saved/quad_chill/Bar.json"} {
		if !IsPresetPath(name) {
			t.Errorf("expected %q to be a preset path", name)
		}
	}
	for _, name := range []string{"", "config/paramdefs.json", "bin/palette.exe", "savedstuff/x.json"} {
		if IsPresetPath(name) {
			t.Errorf("expected %q not to be a preset path", name)
		}
	}
}

func TestPackPreservesPresetModTime(t *testing.T) {
	tmp := t.TempDir()
	stub := filepath.Join(tmp, "stub.exe")
	source := filepath.Join(tmp, "source")
	output := filepath.Join(tmp, "installer.exe")
	if err := os.WriteFile(stub, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "saved", "quad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	presetPath := filepath.Join(source, "saved", "quad", "Foo.json")
	configPath := filepath.Join(source, "config", "paramdefs.json")
	if err := os.WriteFile(presetPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2021, 6, 15, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(presetPath, want, want); err != nil {
		t.Fatal(err)
	}

	if err := Pack(stub, source, output, Manifest{Kind: "data", Version: "1", DataName: "default"}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, _ := f.Stat()
	bundle, err := Read(f, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	var checkedPreset, checkedConfig bool
	for _, entry := range bundle.Archive.File {
		switch entry.Name {
		case "saved/quad/Foo.json":
			checkedPreset = true
			if !entry.Modified.Equal(want) {
				t.Errorf("preset mod time = %v, want %v", entry.Modified.UTC(), want)
			}
		case "config/paramdefs.json":
			checkedConfig = true
			if entry.Modified.UTC().Year() != 1970 {
				t.Errorf("non-preset mod time = %v, want a fixed epoch time", entry.Modified.UTC())
			}
		}
	}
	if !checkedPreset || !checkedConfig {
		t.Fatalf("did not find both entries (preset=%v config=%v)", checkedPreset, checkedConfig)
	}
}

func TestSafeRelativePath(t *testing.T) {
	for _, name := range []string{"bin/palette.exe", "ffgl/Palette.dll"} {
		if !SafeRelativePath(name) {
			t.Errorf("expected %q to be safe", name)
		}
	}
	for _, name := range []string{"", "../outside", "bin/../../outside", `/absolute`, `C:/outside`, `bin\\outside`} {
		if SafeRelativePath(name) {
			t.Errorf("expected %q to be unsafe", name)
		}
	}
}
