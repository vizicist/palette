package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// Replacing an existing file must publish just the new contents.
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

// Slider requests and all-patch changes can save the same working preset at
// once. Every save must succeed and publish a complete document.
func TestWriteFileAtomicConcurrentWriters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_Current.json")
	if err := WriteFileAtomic(path, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	const writers, rounds = 8, 25
	start := make(chan struct{})
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for writer := 0; writer < writers; writer++ {
		wg.Add(1)
		go func(writer int) {
			defer wg.Done()
			<-start
			for round := 0; round < rounds; round++ {
				data := []byte(strings.Repeat(fmt.Sprintf("writer %d round %d\n", writer, round), 100))
				if err := WriteFileAtomic(path, data, 0644); err != nil {
					errs <- err
					return
				}
			}
		}(writer)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 100 {
		t.Fatalf("incomplete save: got %d lines, want 100", len(lines))
	}
	for _, line := range lines {
		if line != lines[0] {
			t.Fatal("saved file contains data from different writers")
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("save left %d files, want just the working preset", len(entries))
	}
}
