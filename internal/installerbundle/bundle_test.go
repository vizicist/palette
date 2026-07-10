package installerbundle

import (
	"io"
	"os"
	"path/filepath"
	"testing"
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
