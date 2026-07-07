package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTarGz builds an in-memory tar.gz with the given files map (path → content).
func makeTarGz(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(content))})
		_, _ = tw.Write([]byte(content))
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return &buf
}

func TestExtractTarGz_Basic(t *testing.T) {
	archive := makeTarGz(t, map[string]string{
		"secrets/prod/api/token.yaml": "data: hello",
		"secrets/prod/db/pass.yaml":   "data: world",
	})

	dir := t.TempDir()
	require.NoError(t, extractTarGz(archive, dir))

	got, err := os.ReadFile(filepath.Join(dir, "secrets", "prod", "api", "token.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "data: hello", string(got))
}

func TestExtractTarGz_RejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "../../evil.txt", Mode: 0o600, Size: 3})
	_, _ = tw.Write([]byte("bad"))
	_ = tw.Close()
	_ = gz.Close()

	dir := t.TempDir()
	err := extractTarGz(&buf, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path")
}

// TestExtractTarGz_AllowsDotDotSubstringInFilename is the L-6 regression
// test: a legitimate filename that merely contains ".." as a substring (not
// a path traversal — no path separator involved) must be extracted, not
// rejected. Path traversal is still caught by TestExtractTarGz_RejectsTraversal.
func TestExtractTarGz_AllowsDotDotSubstringInFilename(t *testing.T) {
	archive := makeTarGz(t, map[string]string{
		"secrets/prod/foo..bar.yaml": "data: hello",
	})

	dir := t.TempDir()
	require.NoError(t, extractTarGz(archive, dir))

	got, err := os.ReadFile(filepath.Join(dir, "secrets", "prod", "foo..bar.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "data: hello", string(got))
}

func TestExtractTarGz_RejectsTraversalViaJoinedSubpath(t *testing.T) {
	// A traversal attempt buried inside an otherwise-plausible path, not just
	// a bare "../../evil.txt" at the top.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "secrets/prod/../../../etc/passwd", Mode: 0o600, Size: 3})
	_, _ = tw.Write([]byte("bad"))
	_ = tw.Close()
	_ = gz.Close()

	dir := t.TempDir()
	err := extractTarGz(&buf, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path")
}

func TestExtractTarGz_SkipsSymlinks(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// Regular file
	_ = tw.WriteHeader(&tar.Header{Name: "ok.yaml", Mode: 0o600, Size: 4})
	_, _ = tw.Write([]byte("data"))
	// Symlink — should be silently skipped
	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     "link.yaml",
		Linkname: "/etc/passwd",
	})
	_ = tw.Close()
	_ = gz.Close()

	dir := t.TempDir()
	require.NoError(t, extractTarGz(&buf, dir))

	_, err := os.Stat(filepath.Join(dir, "link.yaml"))
	assert.True(t, os.IsNotExist(err), "symlink must not be extracted")

	_, err = os.Stat(filepath.Join(dir, "ok.yaml"))
	assert.NoError(t, err, "regular file must be extracted")
}

func TestExtractTarGz_SizeLimit(t *testing.T) {
	// Build an archive that exceeds maxBundleSize when accumulated.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	bigContent := make([]byte, maxBundleSize+1)
	_ = tw.WriteHeader(&tar.Header{Name: "big.yaml", Mode: 0o600, Size: int64(len(bigContent))})
	_, _ = tw.Write(bigContent)
	_ = tw.Close()
	_ = gz.Close()

	dir := t.TempDir()
	err := extractTarGz(&buf, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum size")
}
