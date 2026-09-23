package main

import (
	"os"
	"path/filepath"
	"testing"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
)

// overrideConfigPath redirects configFilePath to a temp directory for the
// duration of the test. Restores the real home via t.Setenv.
func overrideConfigPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return filepath.Join(dir, ".config", "signet", "config.yaml")
}

func TestReadCliConfig_MissingFile(t *testing.T) {
	overrideConfigPath(t)
	cfg, err := readCliConfig()
	if err != nil {
		t.Fatalf("expected no error for missing config file, got %v", err)
	}
	if cfg.Server != "" || cfg.TokenFile != "" {
		t.Errorf("expected zero config, got %+v", cfg)
	}
}

func TestWriteReadCliConfig_RoundTrip(t *testing.T) {
	overrideConfigPath(t)
	want := cliConfig{Server: "localhost:9999", TokenFile: "/tmp/tok"}
	if err := writeCliConfig(want); err != nil {
		t.Fatalf("writeCliConfig: %v", err)
	}
	got, err := readCliConfig()
	if err != nil {
		t.Fatalf("readCliConfig: %v", err)
	}
	if got.Server != want.Server {
		t.Errorf("Server = %q, want %q", got.Server, want.Server)
	}
	if got.TokenFile != want.TokenFile {
		t.Errorf("TokenFile = %q, want %q", got.TokenFile, want.TokenFile)
	}
}

func TestWriteCliConfig_CreatesDirectory(t *testing.T) {
	path := overrideConfigPath(t)
	if err := writeCliConfig(cliConfig{Server: "host:1234"}); err != nil {
		t.Fatalf("writeCliConfig: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestWriteCliConfig_PermissionsRestricted(t *testing.T) {
	path := overrideConfigPath(t)
	if err := writeCliConfig(cliConfig{Server: "host:1234"}); err != nil {
		t.Fatalf("writeCliConfig: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file perm = %o, want 0600", perm)
	}
}

func TestReadCliConfig_InvalidYAML(t *testing.T) {
	path := overrideConfigPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not: valid: yaml: [[["), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := readCliConfig()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// --- resolveToken ---

func TestResolveToken_FromFlag(t *testing.T) {
	tok, err := resolveToken("my-token", "", cliConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "my-token" {
		t.Errorf("got %q, want my-token", tok)
	}
}

func TestResolveToken_FromFlag_TrimsWhitespace(t *testing.T) {
	tok, err := resolveToken("  my-token\n", "", cliConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "my-token" {
		t.Errorf("got %q, want my-token", tok)
	}
}

func TestResolveToken_FromFlagFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(f, []byte("file-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := resolveToken("", f, cliConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "file-token" {
		t.Errorf("got %q, want file-token", tok)
	}
}

func TestResolveToken_FromConfigTokenFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(f, []byte("config-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := resolveToken("", "", cliConfig{TokenFile: f})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "config-token" {
		t.Errorf("got %q, want config-token", tok)
	}
}

func TestResolveToken_FlagTakesPrecedenceOverFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(f, []byte("file-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := resolveToken("flag-token", f, cliConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "flag-token" {
		t.Errorf("got %q, want flag-token", tok)
	}
}

func TestResolveToken_NoToken_ReturnsError(t *testing.T) {
	_, err := resolveToken("", "", cliConfig{})
	if err == nil {
		t.Fatal("expected error when no token is provided")
	}
}

func TestResolveToken_MissingFile_ReturnsError(t *testing.T) {
	_, err := resolveToken("", "/nonexistent/token", cliConfig{})
	if err == nil {
		t.Fatal("expected error for missing token file")
	}
}

func TestResolveToken_EmptyFile_ReturnsError(t *testing.T) {
	f := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(f, []byte("   \n  "), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resolveToken("", f, cliConfig{})
	if err == nil {
		t.Fatal("expected error for empty token file")
	}
}

// --- stateName ---

func TestStateName(t *testing.T) {
	cases := []struct {
		state    adminv1.StatusResponse_State
		expected string
	}{
		{adminv1.StatusResponse_STATE_SEALED, "SEALED"},
		{adminv1.StatusResponse_STATE_UNSEALING, "UNSEALING"},
		{adminv1.StatusResponse_STATE_UNSEALED, "UNSEALED"},
		{adminv1.StatusResponse_STATE_UNSPECIFIED, "UNKNOWN"},
	}
	for _, tc := range cases {
		if got := stateName(tc.state); got != tc.expected {
			t.Errorf("stateName(%v) = %q, want %q", tc.state, got, tc.expected)
		}
	}
}
