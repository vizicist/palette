package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicCreates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.json")

	if err := WriteFileAtomic(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("read back %q, want %q", got, "hello")
	}
}

// Replacing an existing file is the case that matters, and the one that needs
// the extra step on Windows, where rename won't overwrite.
func TestWriteFileAtomicReplaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.json")
	if err := os.WriteFile(path, []byte("old contents, longer than the new"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(path, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("read back %q, want %q", got, "new")
	}
}

// Rewriting the same file over and over is what the feedback database does -
// every Like or Avoid press - so nothing may be left behind.
func TestWriteFileAtomicLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.json")

	for i := 0; i < 10; i++ {
		if err := WriteFileAtomic(path, []byte("contents"), 0644); err != nil {
			t.Fatalf("WriteFileAtomic %d: %v", i, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Fatalf("left a temporary file behind: %s", entry.Name())
		}
	}
	if len(entries) != 1 {
		t.Fatalf("directory holds %d files, want 1", len(entries))
	}
}

// A write into a directory that isn't there must fail rather than half-succeed.
func TestWriteFileAtomicMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "db.json")

	if err := WriteFileAtomic(path, []byte("x"), 0644); err == nil {
		t.Fatal("writing into a missing directory should fail")
	}
	if PathExists(path) {
		t.Fatal("a failed write should leave nothing behind")
	}
}
