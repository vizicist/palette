//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
