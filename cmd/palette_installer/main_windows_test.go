//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseOptions(t *testing.T) {
	opts, err := parseOptions([]string{"--quiet", "--install-root", `C:\\Palette Test`, "/CURRENTUSER"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.quiet || opts.installRoot != `C:\\Palette Test` {
		t.Fatalf("unexpected options: %#v", opts)
	}
	if _, err := parseOptions([]string{"--unknown"}); err == nil {
		t.Fatal("unknown option was accepted")
	}
}

func TestExtractPayload(t *testing.T) {
	var payload bytes.Buffer
	zw := zip.NewWriter(&payload)
	dir, _ := zw.Create("logs/")
	_, _ = dir.Write(nil)
	file, _ := zw.Create("bin/palette.exe")
	_, _ = file.Write([]byte("palette"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(payload.Bytes()), int64(payload.Len()))
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "install")
	files, dirs, err := extractPayload(zr, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || len(dirs) != 2 {
		t.Fatalf("files=%v dirs=%v", files, dirs)
	}
	got, err := os.ReadFile(filepath.Join(root, "bin", "palette.exe"))
	if err != nil || string(got) != "palette" {
		t.Fatalf("installed payload = %q, %v", got, err)
	}
	if info, err := os.Stat(filepath.Join(root, "logs")); err != nil || !info.IsDir() {
		t.Fatal("empty payload directory was not installed")
	}
}

func presetZip(t *testing.T, name, content string, modTime time.Time) *zip.Reader {
	t.Helper()
	var payload bytes.Buffer
	zw := zip.NewWriter(&payload)
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.Modified = modTime
	w, err := zw.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(payload.Bytes()), int64(payload.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return zr
}

func TestExtractPayloadKeepsNewerInstalledPreset(t *testing.T) {
	root := filepath.Join(t.TempDir(), "install")
	const presetRel = "saved/quad/Foo.json"
	dest := filepath.Join(root, filepath.FromSlash(presetRel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("user version"), 0o644); err != nil {
		t.Fatal(err)
	}
	userTime := time.Now()
	if err := os.Chtimes(dest, userTime, userTime); err != nil {
		t.Fatal(err)
	}

	// The bundled preset is older than the user's copy.
	zr := presetZip(t, presetRel, "bundled version", userTime.Add(-48*time.Hour))
	if _, _, err := extractPayload(zr, root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "user version" {
		t.Fatalf("installer overwrote a newer user preset: got %q (%v)", got, err)
	}
}

func TestExtractPayloadOverwritesOlderInstalledPreset(t *testing.T) {
	root := filepath.Join(t.TempDir(), "install")
	const presetRel = "saved/quad/Foo.json"
	dest := filepath.Join(root, filepath.FromSlash(presetRel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old version"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-72 * time.Hour)
	if err := os.Chtimes(dest, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// The bundled preset is newer than the installed copy.
	zr := presetZip(t, presetRel, "bundled version", time.Now().Add(-1*time.Hour))
	if _, _, err := extractPayload(zr, root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "bundled version" {
		t.Fatalf("installer did not update an older preset: got %q (%v)", got, err)
	}
}

func TestExtractPayloadRejectsTraversal(t *testing.T) {
	var payload bytes.Buffer
	zw := zip.NewWriter(&payload)
	file, _ := zw.Create("../outside.txt")
	_, _ = file.Write([]byte("no"))
	_ = zw.Close()
	zr, _ := zip.NewReader(bytes.NewReader(payload.Bytes()), int64(payload.Len()))
	if _, _, err := extractPayload(zr, filepath.Join(t.TempDir(), "install")); err == nil {
		t.Fatal("path traversal payload was accepted")
	}
}

func TestSplitPathDeduplicatesCaseInsensitively(t *testing.T) {
	got := splitPath(`C:\\One;C:\\Two;c:\\one;;`)
	if len(got) != 2 || got[0] != `C:\\One` || got[1] != `C:\\Two` {
		t.Fatalf("splitPath = %#v", got)
	}
}
