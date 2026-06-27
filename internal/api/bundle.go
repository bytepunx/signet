package api

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// maxBundleSize caps the total decompressed bytes accepted from a bundle to
	// guard against decompression bombs from authenticated (but possibly
	// compromised) callers.
	maxBundleSize = 256 << 20 // 256 MiB
)

// extractTarGz decompresses and extracts a tar.gz stream into dir.
// Only regular files are extracted; symlinks and special files are skipped.
// Paths that escape dir via ".." components are rejected.
func extractTarGz(r io.Reader, dir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var total int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		// Only extract regular files; skip directories, symlinks, devices, etc.
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != 0 {
			continue
		}

		// Reject any path with ".." components.
		clean := filepath.Clean(hdr.Name)
		if strings.Contains(clean, "..") {
			return fmt.Errorf("path traversal in archive: %q", hdr.Name)
		}

		dest := filepath.Join(dir, clean)

		// Ensure the final path stays within dir.
		if !strings.HasPrefix(dest, filepath.Clean(dir)+string(os.PathSeparator)) {
			return fmt.Errorf("path escapes extraction dir: %q", hdr.Name)
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
		}

		f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("create %s: %w", dest, err)
		}

		n, copyErr := io.Copy(f, io.LimitReader(tr, maxBundleSize-total+1))
		f.Close()
		if copyErr != nil {
			return fmt.Errorf("write %s: %w", dest, copyErr)
		}
		total += n
		if total > maxBundleSize {
			return fmt.Errorf("bundle exceeds maximum size (%d MiB)", maxBundleSize>>20)
		}
	}
	return nil
}
