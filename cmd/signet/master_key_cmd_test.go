package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRotateTestCmd builds a standalone *cobra.Command with only the flags
// runMasterKeyRotate reads, independent of the package-level masterKeyRotateCmd
// (which must not be mutated/re-registered across tests).
func newRotateTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "rotate"}
	cmd.Flags().String("new-key-file", "", "")
	cmd.Flags().Bool("yes", false, "")
	return cmd
}

func TestRunMasterKeyRotate_InvalidKeyFileSize(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.bin")
	require.NoError(t, os.WriteFile(keyPath, []byte("too-short"), 0o600))

	cmd := newRotateTestCmd()
	require.NoError(t, cmd.Flags().Set("new-key-file", keyPath))
	require.NoError(t, cmd.Flags().Set("yes", "true"))

	var out bytes.Buffer
	err := runMasterKeyRotate(cmd, strings.NewReader(""), &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestRunMasterKeyRotate_MissingKeyFile(t *testing.T) {
	cmd := newRotateTestCmd()
	require.NoError(t, cmd.Flags().Set("new-key-file", "/nonexistent/path/key.bin"))

	var out bytes.Buffer
	err := runMasterKeyRotate(cmd, strings.NewReader(""), &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read --new-key-file")
}

func TestRunMasterKeyRotate_DeclinedConfirmationNeverDials(t *testing.T) {
	// With a generated key and a declined confirmation, the function must
	// return before attempting to dial the admin server — otherwise this
	// test would hang or fail on a connection error instead of the expected
	// confirmation error.
	cmd := newRotateTestCmd()

	var out bytes.Buffer
	err := runMasterKeyRotate(cmd, strings.NewReader("no\n"), &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
}
