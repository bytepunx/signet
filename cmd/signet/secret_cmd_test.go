package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/bytepunx/signet/internal/gitops"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitSecretRef(t *testing.T) {
	ns, svc, name, err := splitSecretRef("payments/api/stripe-key")
	require.NoError(t, err)
	assert.Equal(t, "payments", ns)
	assert.Equal(t, "api", svc)
	assert.Equal(t, "stripe-key", name)

	for _, bad := range []string{"payments/api", "payments/api/stripe-key/extra", "//name", "payments//name", ""} {
		_, _, _, err := splitSecretRef(bad)
		assert.Error(t, err, "expected error for %q", bad)
	}
}

func TestFindSOPSConfig(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".sops.yaml"), []byte("{}"), 0o644))
	deep := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deep, 0o755))

	repoRoot, configPath, err := findSOPSConfig(deep)
	require.NoError(t, err)
	assert.Equal(t, root, repoRoot)
	assert.Equal(t, filepath.Join(root, ".sops.yaml"), configPath)
}

func TestFindSOPSConfig_NotFound(t *testing.T) {
	// A temp dir has no .sops.yaml anywhere above it (assuming the OS temp
	// root itself doesn't have one, which it never does in CI/dev).
	_, _, err := findSOPSConfig(t.TempDir())
	assert.Error(t, err)
}

func TestResolveEnvironment(t *testing.T) {
	writeConfig := func(t *testing.T, body string) string {
		path := filepath.Join(t.TempDir(), ".sops.yaml")
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		return path
	}

	t.Run("explicit flag wins", func(t *testing.T) {
		path := writeConfig(t, `environments:
    prod: age1prod111
    staging: age1staging222
`)
		env, auto, err := resolveEnvironment(path, "staging")
		require.NoError(t, err)
		assert.Equal(t, "staging", env)
		assert.False(t, auto)
	})

	t.Run("no environments means global", func(t *testing.T) {
		path := writeConfig(t, `creation_rules:
    - path_regex: ^secrets/
      age: age1global000
`)
		env, auto, err := resolveEnvironment(path, "")
		require.NoError(t, err)
		assert.Equal(t, "", env)
		assert.False(t, auto)
	})

	t.Run("single environment auto-selected", func(t *testing.T) {
		path := writeConfig(t, `environments:
    prod: age1prod111
`)
		env, auto, err := resolveEnvironment(path, "")
		require.NoError(t, err)
		assert.Equal(t, "prod", env)
		assert.True(t, auto)
	})

	t.Run("multiple environments require --env", func(t *testing.T) {
		path := writeConfig(t, `environments:
    prod: age1prod111
    staging: age1staging222
`)
		_, _, err := resolveEnvironment(path, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prod")
		assert.Contains(t, err.Error(), "staging")
	})
}

func TestResolveSecretPath(t *testing.T) {
	assert.Equal(t, "secrets/payments/api/stripe-key.yaml",
		resolveSecretPath("secrets/", "", "payments", "api", "stripe-key"))
	assert.Equal(t, "secrets/prod/payments/api/stripe-key.yaml",
		resolveSecretPath("secrets/", "prod", "payments", "api", "stripe-key"))
}

func testCmd() *cobra.Command {
	c := &cobra.Command{Use: "test"}
	c.SetOut(new(strings.Builder))
	return c
}

func TestResolveSecretValue(t *testing.T) {
	t.Run("value flag wins", func(t *testing.T) {
		v, err := resolveSecretValue(testCmd(), strings.NewReader(""), "flagval", "")
		require.NoError(t, err)
		assert.Equal(t, "flagval", v)
	})

	t.Run("value and value-file mutually exclusive", func(t *testing.T) {
		_, err := resolveSecretValue(testCmd(), strings.NewReader(""), "a", "b")
		assert.Error(t, err)
	})

	t.Run("value-file is read and trimmed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "value.txt")
		require.NoError(t, os.WriteFile(path, []byte("filesecret\n"), 0o600))
		v, err := resolveSecretValue(testCmd(), strings.NewReader(""), "", path)
		require.NoError(t, err)
		assert.Equal(t, "filesecret", v)
	})

	t.Run("piped stdin is read and trimmed", func(t *testing.T) {
		v, err := resolveSecretValue(testCmd(), strings.NewReader("pipedsecret\n"), "", "")
		require.NoError(t, err)
		assert.Equal(t, "pipedsecret", v)
	})
}

// stubRunSops swaps the package-level runSops var for the duration of the
// test, restoring the original on cleanup.
func stubRunSops(t *testing.T, fn func(dir string, args []string) ([]byte, error)) {
	t.Helper()
	orig := runSops
	runSops = fn
	t.Cleanup(func() { runSops = orig })
}

func TestRunSecretSet_MissingSOPSConfig(t *testing.T) {
	dir := t.TempDir()
	secretSetFlags = struct {
		value       string
		valueFile   string
		env         string
		secretsRoot string
		sopsConfig  string
	}{value: "x", secretsRoot: "secrets/", sopsConfig: filepath.Join(dir, "does-not-exist.yaml")}

	err := runSecretSet(testCmd(), strings.NewReader(""), "ns/svc/name")
	require.Error(t, err)
}

func TestRunSecretSet_WritesAndEncrypts(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".sops.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("creation_rules:\n    - age: age1dummy\n"), 0o644))

	var gotDir string
	var gotArgs []string
	var sawPlaintext []byte
	stubRunSops(t, func(dir string, args []string) ([]byte, error) {
		gotDir = dir
		gotArgs = args
		// args are ["--config", configPath, "--encrypt", "--in-place", relPath]
		relPath := args[len(args)-1]
		data, err := os.ReadFile(filepath.Join(dir, relPath))
		require.NoError(t, err)
		sawPlaintext = data
		return nil, nil
	})

	secretSetFlags = struct {
		value       string
		valueFile   string
		env         string
		secretsRoot string
		sopsConfig  string
	}{value: "sk_live_abc", secretsRoot: "secrets/", sopsConfig: configPath}

	err := runSecretSet(testCmd(), strings.NewReader(""), "payments/api/stripe-key")
	require.NoError(t, err)

	assert.Equal(t, repoRoot, gotDir)
	assert.Equal(t, []string{"--config", configPath, "--encrypt", "--in-place", "secrets/payments/api/stripe-key.yaml"}, gotArgs)
	assert.Contains(t, string(sawPlaintext), "sk_live_abc")
}

func TestRunSecretSet_SopsFailureRemovesPlaintext(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".sops.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("creation_rules: []\n"), 0o644))

	stubRunSops(t, func(dir string, args []string) ([]byte, error) {
		return []byte("sops: no matching creation rule"), fmt.Errorf("exit status 1")
	})

	secretSetFlags = struct {
		value       string
		valueFile   string
		env         string
		secretsRoot string
		sopsConfig  string
	}{value: "sk_live_abc", secretsRoot: "secrets/", sopsConfig: configPath}

	err := runSecretSet(testCmd(), strings.NewReader(""), "payments/api/stripe-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching creation rule")

	_, statErr := os.Stat(filepath.Join(repoRoot, "secrets", "payments", "api", "stripe-key.yaml"))
	assert.True(t, os.IsNotExist(statErr), "plaintext file must be removed after a failed encrypt")
}

func TestRunSops_MissingBinary(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { lookPath = origLookPath })

	_, err := runSops(t.TempDir(), []string{"--encrypt", "--in-place", "x.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sops binary not found")
}

func TestRunSecretRm(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".sops.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("creation_rules: []\n"), 0o644))

	secretPath := filepath.Join(repoRoot, "secrets", "payments", "api", "stripe-key.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(secretPath), 0o755))
	require.NoError(t, os.WriteFile(secretPath, []byte("sops: {}\n"), 0o600))

	secretRmFlags = struct {
		env         string
		secretsRoot string
		sopsConfig  string
	}{secretsRoot: "secrets/", sopsConfig: configPath}

	require.NoError(t, runSecretRm(testCmd(), "payments/api/stripe-key"))
	_, statErr := os.Stat(secretPath)
	assert.True(t, os.IsNotExist(statErr))

	err := runSecretRm(testCmd(), "payments/api/stripe-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no secret at")
}

// TestSecretSet_RealSopsRoundTrip exercises the actual sops binary end to
// end: it writes a real age recipient into .sops.yaml, runs the real
// (non-stubbed) runSops, and confirms the resulting file decrypts back to
// the original value via the same DecryptFile path signetd itself uses.
func TestSecretSet_RealSopsRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("sops binary not found on PATH; skipping real round-trip test")
	}

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	recipient := identity.Recipient().String()

	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".sops.yaml")
	config := fmt.Sprintf("creation_rules:\n    - path_regex: ^secrets/\n      age: %s\n", recipient)
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

	secretSetFlags = struct {
		value       string
		valueFile   string
		env         string
		secretsRoot string
		sopsConfig  string
	}{value: "correct-horse-battery-staple", secretsRoot: "secrets/", sopsConfig: configPath}

	require.NoError(t, runSecretSet(testCmd(), strings.NewReader(""), "payments/api/stripe-key"))

	encrypted, err := os.ReadFile(filepath.Join(repoRoot, "secrets", "payments", "api", "stripe-key.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(encrypted), "ENC[") // sanity check it's actually encrypted

	plain, err := gitops.DecryptFile(encrypted, []age.Identity{identity})
	require.NoError(t, err)
	assert.Equal(t, "correct-horse-battery-staple", string(plain))
}
