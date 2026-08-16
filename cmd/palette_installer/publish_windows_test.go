//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// zipWith builds an in-memory payload archive.
func zipWith(t *testing.T, files map[string]string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.Modified = time.Now()
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return zr
}

// A normal install replaces the previous files and leaves no scaffolding
// behind - in particular no .palette-old copies.
func TestExtractPayloadReplacesAndCleansUp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, _, err := extractPayload(zipWith(t, map[string]string{
		"a.txt": "new",
		"b.txt": "brand new",
	}), root)
	if err != nil {
		t.Fatalf("extractPayload: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("recorded %d files, want 2", len(files))
	}

	got, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	if string(got) != "new" {
		t.Errorf("a.txt is %q, want the new contents", got)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt.palette-old")); !os.IsNotExist(err) {
		t.Error("a superseded copy was left behind after a successful install")
	}
}

// If publishing fails part way through, the previous version has to come back.
// A live file used to be deleted and then replaced, so a failure left that file
// missing and the directory a mixture of old, new and absent files.
func TestExtractPayloadRollsBackOnFailure(t *testing.T) {
	root := t.TempDir()

	// Two live files that the payload will replace.
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("old "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A third live file held open with no sharing at all, which is what an
	// antivirus scanner or a still-running process does: Windows then refuses
	// to rename it, and this is the failure the rollback exists for. Go's own
	// os.OpenFile shares delete access, so it would not reproduce this.
	blocker := filepath.Join(root, "c.txt")
	if err := os.WriteFile(blocker, []byte("old c.txt"), 0o644); err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(
		syscall.StringToUTF16Ptr(blocker),
		syscall.GENERIC_READ,
		0, // exclusive: no read, write or delete sharing
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0)
	if err != nil {
		t.Fatalf("could not take an exclusive handle on %s: %v", blocker, err)
	}
	defer syscall.CloseHandle(handle)

	_, _, err = extractPayload(zipWith(t, map[string]string{
		"a.txt": "new a",
		"b.txt": "new b",
		"c.txt": "new c",
	}), root)
	if err == nil {
		t.Fatal("expected publishing to fail when a destination cannot be replaced")
	}

	// Whatever order the files went in, every pre-existing one must be intact.
	for _, name := range []string{"a.txt", "b.txt"} {
		got, readErr := os.ReadFile(filepath.Join(root, name))
		if readErr != nil {
			t.Fatalf("%s is missing after a failed install: %v", name, readErr)
		}
		if string(got) != "old "+name {
			t.Errorf("%s is %q, want its previous contents back", name, got)
		}
	}
	// And no scaffolding left over.
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".palette-old" {
			t.Errorf("rollback left %s behind", e.Name())
		}
	}
}
