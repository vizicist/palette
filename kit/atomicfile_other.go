//go:build !windows

package kit

import "os"

func replaceFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
