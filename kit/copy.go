package engine

import (
	"io"
	"os"
	"path/filepath"
)

// CopyDir xxx
func CopyDir(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	return copy(src, dest, info)
}

// copy dispatches copy-funcs according to the mode.
// Because this "copy" could be called recursively,
// "info" MUST be given here, NOT nil.
func copy(src, dest string, info os.FileInfo) error {

	if info.IsDir() {
		return dcopy(src, dest)
	}
	return fcopy(src, dest, info)
}

// fcopy is for just a file,
// with considering existence of parent directory
// and file permission.
func fcopy(src, dest string, info os.FileInfo) error {

	err := os.MkdirAll(filepath.Dir(dest), os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fclose(f, &err)

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fclose(s, &err)

	_, err = io.Copy(f, s)
	if err != nil {
		return err
	}

	return f.Sync() // needed?
}

// dcopy is for a directory,
// with scanning contents inside the directory
// and pass everything to "copy" recursively.
func dcopy(srcdir, destdir string) (err error) {

	if err = os.MkdirAll(destdir, os.FileMode(0755)); err != nil {
		return
	}

	entries, err := os.ReadDir(srcdir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		einfo, err2 := entry.Info()
		if err2 != nil {
			LogWarn("dcopy", "err", err2)
			continue
		}
		name := einfo.Name()
		cs, cd := filepath.Join(srcdir, name), filepath.Join(destdir, name)
		if err = copy(cs, cd, einfo); err != nil {
			// If any error, exit immediately
			return
		}
	}

	return
}

// fclose ANYHOW closes file,
// with asiging error raised during Close,
// BUT respecting the error already reported.
func fclose(f *os.File, reported *error) {
	if err := f.Close(); *reported == nil {
		*reported = err
	}
}
