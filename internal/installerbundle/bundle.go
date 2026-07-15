// Package installerbundle builds and reads Palette's self-contained installer
// format. An installer is a native executable followed by a ZIP archive, a JSON
// manifest, and a fixed-size footer describing those sections.
package installerbundle

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const footerSize = 40

var footerMagic = [16]byte{'P', 'A', 'L', 'E', 'T', 'T', 'E', '-', 'I', 'N', 'S', 'T', 'A', 'L', 'L', '1'}

// Manifest describes how the payload is installed.
type Manifest struct {
	Kind     string   `json:"kind"`
	Version  string   `json:"version"`
	DataName string   `json:"data_name,omitempty"`
	Delete   []string `json:"delete,omitempty"`
}

// Bundle is a parsed installer bundle.
type Bundle struct {
	Manifest Manifest
	Archive  *zip.Reader
	StubSize int64
}

// Pack creates a self-contained installer from a native stub and source tree.
func Pack(stubPath, sourcePath, outputPath string, manifest Manifest) error {
	if err := validateManifest(manifest); err != nil {
		return err
	}
	stub, err := os.Open(stubPath)
	if err != nil {
		return fmt.Errorf("open installer stub: %w", err)
	}
	defer stub.Close()

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), filepath.Base(outputPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create installer: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	stubSize, err := io.Copy(tmp, stub)
	if err != nil {
		return fmt.Errorf("copy installer stub: %w", err)
	}
	zipStart := stubSize
	zw := zip.NewWriter(tmp)
	if err := addTree(zw, sourcePath); err != nil {
		zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("finish payload archive: %w", err)
	}
	zipEnd, err := tmp.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("locate payload archive: %w", err)
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if _, err := tmp.Write(manifestJSON); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	footer := make([]byte, footerSize)
	copy(footer[:16], footerMagic[:])
	binary.LittleEndian.PutUint64(footer[16:24], uint64(zipStart))
	binary.LittleEndian.PutUint64(footer[24:32], uint64(zipEnd-zipStart))
	binary.LittleEndian.PutUint64(footer[32:40], uint64(len(manifestJSON)))
	if _, err := tmp.Write(footer); err != nil {
		return fmt.Errorf("write footer: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("flush installer: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close installer: %w", err)
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return fmt.Errorf("mark installer executable: %w", err)
	}
	if err := os.Remove(outputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace old installer: %w", err)
	}
	if err := os.Rename(tmpName, outputPath); err != nil {
		return fmt.Errorf("publish installer: %w", err)
	}
	return nil
}

// Read parses an installer from an io.ReaderAt.
func Read(r io.ReaderAt, size int64) (*Bundle, error) {
	if size < footerSize {
		return nil, errors.New("file does not contain a Palette installer payload")
	}
	footer := make([]byte, footerSize)
	if _, err := r.ReadAt(footer, size-footerSize); err != nil {
		return nil, fmt.Errorf("read installer footer: %w", err)
	}
	if !bytes.Equal(footer[:16], footerMagic[:]) {
		return nil, errors.New("file does not contain a Palette installer payload")
	}
	zipStartValue := binary.LittleEndian.Uint64(footer[16:24])
	zipSizeValue := binary.LittleEndian.Uint64(footer[24:32])
	manifestSizeValue := binary.LittleEndian.Uint64(footer[32:40])
	contentEnd := uint64(size - footerSize)
	if manifestSizeValue == 0 || manifestSizeValue > contentEnd {
		return nil, errors.New("installer footer contains invalid offsets")
	}
	manifestStartValue := contentEnd - manifestSizeValue
	if zipStartValue > manifestStartValue || zipSizeValue == 0 || zipSizeValue != manifestStartValue-zipStartValue {
		return nil, errors.New("installer footer contains invalid offsets")
	}
	zipStart := int64(zipStartValue)
	zipSize := int64(zipSizeValue)
	manifestSize := int64(manifestSizeValue)
	manifestStart := int64(manifestStartValue)
	manifestJSON := make([]byte, manifestSize)
	if _, err := r.ReadAt(manifestJSON, manifestStart); err != nil {
		return nil, fmt.Errorf("read installer manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return nil, fmt.Errorf("decode installer manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(io.NewSectionReader(r, zipStart, zipSize), zipSize)
	if err != nil {
		return nil, fmt.Errorf("open installer payload: %w", err)
	}
	return &Bundle{Manifest: manifest, Archive: zr, StubSize: zipStart}, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Kind != "app" && manifest.Kind != "data" {
		return fmt.Errorf("invalid installer kind %q", manifest.Kind)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return errors.New("installer version is required")
	}
	if manifest.Kind == "data" && strings.TrimSpace(manifest.DataName) == "" {
		return errors.New("data installer name is required")
	}
	if strings.ContainsAny(manifest.DataName, `\/`) {
		return errors.New("data installer name must not contain path separators")
	}
	for _, name := range manifest.Delete {
		if !SafeRelativePath(name) {
			return fmt.Errorf("unsafe delete path %q", name)
		}
	}
	return nil
}

func addTree(zw *zip.Writer, root string) error {
	var names []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("installer payload cannot contain symlink %q", path)
		}
		names = append(names, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan payload: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		info, err := os.Stat(name)
		if err != nil {
			return fmt.Errorf("stat payload file: %w", err)
		}
		rel, err := filepath.Rel(root, name)
		if err != nil || !SafeRelativePath(filepath.ToSlash(rel)) {
			return fmt.Errorf("unsafe payload path %q", name)
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("describe payload file: %w", err)
		}
		header.Name = filepath.ToSlash(rel)
		if IsPresetPath(header.Name) {
			// Preserve the real modification time for preset files so the
			// installer can avoid overwriting presets the user has modified
			// since installing. Other files use a fixed time for reproducible,
			// deterministic installer output.
			header.Modified = info.ModTime().UTC()
		} else {
			header.Modified = time.Unix(0, 0).UTC()
		}
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		out, err := zw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("add payload file %q: %w", rel, err)
		}
		if info.IsDir() {
			continue
		}
		in, err := os.Open(name)
		if err != nil {
			return fmt.Errorf("open payload file %q: %w", rel, err)
		}
		_, copyErr := io.Copy(out, in)
		closeErr := in.Close()
		if copyErr != nil {
			return fmt.Errorf("compress payload file %q: %w", rel, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close payload file %q: %w", rel, closeErr)
		}
	}
	return nil
}

// IsPresetPath reports whether a payload-relative path is a user preset file,
// i.e. one under the saved/ directory. The installer must not overwrite such a
// file when the installed copy is newer than the bundled one, so these files
// carry their real modification time in the payload.
func IsPresetPath(name string) bool {
	name = filepath.ToSlash(name)
	return name == "saved" || strings.HasPrefix(name, "saved/")
}

// SafeRelativePath reports whether a slash-separated archive path stays below
// its extraction root.
func SafeRelativePath(name string) bool {
	if name == "" || strings.ContainsRune(name, '\x00') || strings.Contains(name, `\`) {
		return false
	}
	if strings.HasPrefix(name, "/") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(name)))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && !filepath.IsAbs(name) && !strings.Contains(clean, ":")
}
