package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDecryptFile_RealSopsCiphertext regression-tests DecryptFile against a
// file produced by the real sops binary. This previously failed:
// tree.Metadata.GetDataKey() routes decryption through sops's local
// keyservice, which reconstructs a fresh age.MasterKey from just the
// wire-serialized recipient string rather than using the *MasterKey objects
// that had identities injected via ParsedIdentities.ApplyToMasterKey — so
// the injected identities were silently discarded and decryption fell back
// to looking for SOPS_AGE_KEY / ~/.ssh/id_rsa, none of which signet sets.
func TestDecryptFile_RealSopsCiphertext(t *testing.T) {
	sopsPath, err := exec.LookPath("sops")
	if err != nil {
		t.Skip("sops binary not found on PATH; skipping real ciphertext regression test")
	}

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sops.yaml")
	config := fmt.Sprintf("creation_rules:\n    - path_regex: ^secrets/\n      age: %s\n", identity.Recipient().String())
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

	secretsDir := filepath.Join(dir, "secrets")
	require.NoError(t, os.MkdirAll(secretsDir, 0o755))
	secretPath := filepath.Join(secretsDir, "test.yaml")
	require.NoError(t, os.WriteFile(secretPath, []byte("value: hello-world\n"), 0o600))

	cmd := exec.Command(sopsPath, "--config", configPath, "--encrypt", "--in-place", "secrets/test.yaml")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "sops encrypt: %s", out)

	encrypted, err := os.ReadFile(secretPath)
	require.NoError(t, err)

	plain, err := DecryptFile(encrypted, []age.Identity{identity})
	require.NoError(t, err)
	assert.Equal(t, "hello-world", string(plain))
}

// TestDecryptFile_WrongIdentityFails verifies that an identity not among the
// file's recipients is rejected rather than silently succeeding.
func TestDecryptFile_WrongIdentityFails(t *testing.T) {
	sopsPath, err := exec.LookPath("sops")
	if err != nil {
		t.Skip("sops binary not found on PATH; skipping real ciphertext regression test")
	}

	encryptedTo, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	wrongIdentity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sops.yaml")
	config := fmt.Sprintf("creation_rules:\n    - path_regex: ^secrets/\n      age: %s\n", encryptedTo.Recipient().String())
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

	secretsDir := filepath.Join(dir, "secrets")
	require.NoError(t, os.MkdirAll(secretsDir, 0o755))
	secretPath := filepath.Join(secretsDir, "test.yaml")
	require.NoError(t, os.WriteFile(secretPath, []byte("value: hello-world\n"), 0o600))

	cmd := exec.Command(sopsPath, "--config", configPath, "--encrypt", "--in-place", "secrets/test.yaml")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "sops encrypt: %s", out)

	encrypted, err := os.ReadFile(secretPath)
	require.NoError(t, err)

	_, err = DecryptFile(encrypted, []age.Identity{wrongIdentity})
	assert.Error(t, err)
}
