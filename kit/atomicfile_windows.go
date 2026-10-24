package kit

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// os.Rename uses MoveFileEx with MOVEFILE_REPLACE_EXISTING on Windows.
// Readers and other file replacements can briefly prevent deletion/replacement
// of the target, so allow those handles to close without removing the old file.
func replaceFile(oldpath, newpath string) error {
	deadline := time.Now().Add(time.Second)
	for {
		err := os.Rename(oldpath, newpath)
		if err == nil || (!errors.Is(err, windows.ERROR_ACCESS_DENIED) &&
			!errors.Is(err, windows.ERROR_SHARING_VIOLATION) &&
			!errors.Is(err, windows.ERROR_LOCK_VIOLATION)) || !time.Now().Before(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
