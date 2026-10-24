package kit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// A reader (including antivirus/indexing tools) can briefly hold the preset
// without delete sharing. Saving must keep the old preset until that handle
// closes, then replace it rather than report a failed parameter change.
func TestWriteFileAtomicWaitsForWindowsReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "_Current.json")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if handle != windows.InvalidHandle {
			windows.CloseHandle(handle)
		}
	}()
	done := make(chan error, 1)
	go func() { done <- WriteFileAtomic(path, []byte("new"), 0644) }()
	select {
	case err := <-done:
		t.Fatalf("save returned before the reader released its handle: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if old, err := os.ReadFile(path); err != nil || string(old) != "old" {
		t.Fatalf("old preset was lost while replacement was blocked: %q, %v", old, err)
	}
	if err := windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	handle = windows.InvalidHandle
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("save did not finish after the reader released its handle")
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "new" {
		t.Fatalf("replacement preset = %q, %v", got, err)
	}
}
