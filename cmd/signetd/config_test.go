package main

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// validRaw returns a rawConfig with all required fields set to valid values.
func validRaw() rawConfig {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return rawConfig{
		dbConnString:    "postgres://root@localhost:26257/signet",
		spireSocket:     "unix:///run/spire/sockets/agent.sock",
		trustDomain:     "example.org",
		workloadAddr:    ":8443",
		adminAddr:       "127.0.0.1:8444",
		drainTimeout:    "30s",
		shamirShares:    0,
		shamirThreshold: 0,
		shareTimeout:    "30m",
		kubeAudiences:   "signet",
		auditChainKey:   hex.EncodeToString(key),
	}
}

func TestValidate_Success(t *testing.T) {
	cfg, err := validate(validRaw())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TrustDomain != "example.org" {
		t.Errorf("TrustDomain = %q", cfg.TrustDomain)
	}
	if cfg.DrainTimeout != 30*time.Second {
		t.Errorf("DrainTimeout = %v", cfg.DrainTimeout)
	}
	if len(cfg.auditKeyBytes) != 32 {
		t.Errorf("auditKeyBytes len = %d, want 32", len(cfg.auditKeyBytes))
	}
}

func TestValidate_MissingDB(t *testing.T) {
	r := validRaw()
	r.dbConnString = ""
	_, err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "SIGNET_DB_CONN_STRING") {
		t.Errorf("expected error mentioning SIGNET_DB_CONN_STRING, got %v", err)
	}
}

func TestValidate_MissingTrustDomain(t *testing.T) {
	r := validRaw()
	r.trustDomain = ""
	_, err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "SIGNET_TRUST_DOMAIN") {
		t.Errorf("expected error mentioning SIGNET_TRUST_DOMAIN, got %v", err)
	}
}

func TestValidate_MissingAuditChainKey(t *testing.T) {
	r := validRaw()
	r.auditChainKey = ""
	_, err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "SIGNET_AUDIT_CHAIN_KEY") {
		t.Errorf("expected error mentioning SIGNET_AUDIT_CHAIN_KEY, got %v", err)
	}
}

func TestValidate_InvalidAuditChainKeyHex(t *testing.T) {
	r := validRaw()
	r.auditChainKey = "not-valid-hex"
	_, err := validate(r)
	if err == nil {
		t.Fatal("expected error for invalid hex audit chain key")
	}
}

func TestValidate_AuditChainKeyWrongLength(t *testing.T) {
	// 16 bytes = 32 hex chars — too short
	r := validRaw()
	r.auditChainKey = hex.EncodeToString(make([]byte, 16))
	_, err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Errorf("expected error about key length, got %v", err)
	}
}

// TestValidate_AuditChainKeyFromFile is the L-4 regression test: the chain
// key can be supplied via a file instead of inline on the command line/env.
func TestValidate_AuditChainKeyFromFile(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "chain.key")
	if err := os.WriteFile(path, []byte(hex.EncodeToString(key)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := validRaw()
	r.auditChainKey = ""
	r.auditChainKeyFile = path
	cfg, err := validate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.auditKeyBytes) != 32 {
		t.Errorf("auditKeyBytes len = %d, want 32", len(cfg.auditKeyBytes))
	}
	for i, b := range cfg.auditKeyBytes {
		if b != key[i] {
			t.Fatalf("auditKeyBytes mismatch at %d: got %x want %x", i, b, key[i])
		}
	}
}

func TestValidate_AuditChainKeyFileMissing(t *testing.T) {
	r := validRaw()
	r.auditChainKey = ""
	r.auditChainKeyFile = "/nonexistent/path/chain.key"
	_, err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "audit-chain-key-file") {
		t.Errorf("expected error mentioning audit-chain-key-file, got %v", err)
	}
}

func TestValidate_AuditChainKeyBothInlineAndFileRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chain.key")
	if err := os.WriteFile(path, []byte(hex.EncodeToString(make([]byte, 32))), 0o600); err != nil {
		t.Fatal(err)
	}

	r := validRaw() // already sets auditChainKey
	r.auditChainKeyFile = path
	_, err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("expected error requiring exactly one source, got %v", err)
	}
}

func TestValidate_InvalidDrainTimeout(t *testing.T) {
	r := validRaw()
	r.drainTimeout = "not-a-duration"
	_, err := validate(r)
	if err == nil {
		t.Fatal("expected error for invalid drain timeout")
	}
}

func TestValidate_InvalidShareTimeout(t *testing.T) {
	r := validRaw()
	r.shareTimeout = "bad"
	_, err := validate(r)
	if err == nil {
		t.Fatal("expected error for invalid share timeout")
	}
}

func TestValidate_ShamirThresholdTooLow(t *testing.T) {
	r := validRaw()
	r.shamirShares = 3
	r.shamirThreshold = 1 // must be >= 2
	_, err := validate(r)
	if err == nil || !strings.Contains(err.Error(), "threshold") {
		t.Errorf("expected threshold error, got %v", err)
	}
}

func TestValidate_ShamirSharesLessThanThreshold(t *testing.T) {
	r := validRaw()
	r.shamirShares = 2
	r.shamirThreshold = 3
	_, err := validate(r)
	if err == nil {
		t.Fatal("expected error: shares < threshold")
	}
}

func TestValidate_ShamirValid(t *testing.T) {
	r := validRaw()
	r.shamirShares = 5
	r.shamirThreshold = 3
	cfg, err := validate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ShamirShares != 5 || cfg.ShamirThreshold != 3 {
		t.Errorf("shamir = %d/%d, want 5/3", cfg.ShamirShares, cfg.ShamirThreshold)
	}
}

func TestValidate_DirectKeyMode(t *testing.T) {
	r := validRaw()
	r.shamirShares = 0
	r.shamirThreshold = 0
	cfg, err := validate(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ShamirThreshold != 0 {
		t.Errorf("ShamirThreshold = %d, want 0 for direct key mode", cfg.ShamirThreshold)
	}
}

func TestValidate_KubeAudiencesParsed(t *testing.T) {
	r := validRaw()
	r.kubeAudiences = "signet, api-server , other"
	cfg, err := validate(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.kubeAudiences) != 3 {
		t.Errorf("kubeAudiences = %v, want 3 entries", cfg.kubeAudiences)
	}
}

// TestValidate_EmptyKubeAudiencesRejected is the C-2 regression test: an
// empty audience list must be a fatal misconfiguration, not silently widen
// to "accept any audience".
func TestValidate_EmptyKubeAudiencesRejected(t *testing.T) {
	r := validRaw()
	r.kubeAudiences = ""
	_, err := validate(r)
	if err == nil {
		t.Fatal("expected error for empty kube-audiences")
	}
	if !strings.Contains(err.Error(), "kube-audiences") {
		t.Errorf("error should mention kube-audiences: %v", err)
	}
}

// TestValidate_WhitespaceOnlyKubeAudiencesRejected covers the case where the
// raw string is non-empty but contains no usable entries after trimming.
func TestValidate_WhitespaceOnlyKubeAudiencesRejected(t *testing.T) {
	r := validRaw()
	r.kubeAudiences = " , , "
	_, err := validate(r)
	if err == nil {
		t.Fatal("expected error for whitespace-only kube-audiences")
	}
}

func TestValidate_AdminSubjectsParsed(t *testing.T) {
	r := validRaw()
	r.adminSubjects = "serviceaccount:signet:signet-admin, group:signet-operators"
	cfg, err := validate(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.adminSubjects) != 2 {
		t.Errorf("adminSubjects = %v, want 2 entries", cfg.adminSubjects)
	}
}

func TestValidate_AdminSubjectsOptional(t *testing.T) {
	r := validRaw()
	r.adminSubjects = ""
	cfg, err := validate(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.adminSubjects) != 0 {
		t.Errorf("adminSubjects = %v, want empty when unset", cfg.adminSubjects)
	}
}

func TestValidate_AuditFailClosedPassthrough(t *testing.T) {
	r := validRaw()
	r.auditFailClosed = true
	cfg, err := validate(r)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AuditFailClosed {
		t.Error("AuditFailClosed = false, want true")
	}

	r.auditFailClosed = false
	cfg, err = validate(r)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuditFailClosed {
		t.Error("AuditFailClosed = true, want false")
	}
}

func TestEnvOrBool(t *testing.T) {
	t.Setenv("SIGNET_TEST_BOOL_UNSET", "")
	if got := envOrBool("SIGNET_TEST_BOOL_UNSET", true); got != true {
		t.Errorf("unset: got %v, want true (fallback)", got)
	}

	t.Setenv("SIGNET_TEST_BOOL_FALSE", "false")
	if got := envOrBool("SIGNET_TEST_BOOL_FALSE", true); got != false {
		t.Errorf("false: got %v, want false", got)
	}

	t.Setenv("SIGNET_TEST_BOOL_TRUE", "true")
	if got := envOrBool("SIGNET_TEST_BOOL_TRUE", false); got != true {
		t.Errorf("true: got %v, want true", got)
	}

	t.Setenv("SIGNET_TEST_BOOL_INVALID", "not-a-bool")
	if got := envOrBool("SIGNET_TEST_BOOL_INVALID", true); got != true {
		t.Errorf("invalid: got %v, want true (fallback)", got)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	r := validRaw()
	r.dbConnString = ""
	r.trustDomain = ""
	r.auditChainKey = ""
	_, err := validate(r)
	if err == nil {
		t.Fatal("expected multiple errors")
	}
	// All three missing fields should be reported in one error.
	msg := err.Error()
	if !strings.Contains(msg, "SIGNET_DB_CONN_STRING") ||
		!strings.Contains(msg, "SIGNET_TRUST_DOMAIN") ||
		!strings.Contains(msg, "SIGNET_AUDIT_CHAIN_KEY") {
		t.Errorf("expected all three missing-field errors, got:\n%s", msg)
	}
}

func TestEnvOr_UsesEnvWhenSet(t *testing.T) {
	t.Setenv("SIGNET_TEST_KEY", "from-env")
	if got := envOr("SIGNET_TEST_KEY", "default"); got != "from-env" {
		t.Errorf("got %q, want from-env", got)
	}
}

func TestEnvOr_UsesFallbackWhenMissing(t *testing.T) {
	t.Setenv("SIGNET_TEST_KEY_MISSING", "")
	if got := envOr("SIGNET_TEST_KEY_MISSING", "default"); got != "default" {
		t.Errorf("got %q, want default", got)
	}
}

func TestEnvOrInt_ParsesInteger(t *testing.T) {
	t.Setenv("SIGNET_TEST_INT", "42")
	if got := envOrInt("SIGNET_TEST_INT", 0); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestEnvOrInt_UsesFallbackOnBadValue(t *testing.T) {
	t.Setenv("SIGNET_TEST_INT_BAD", "not-a-number")
	if got := envOrInt("SIGNET_TEST_INT_BAD", 7); got != 7 {
		t.Errorf("got %d, want 7 (fallback)", got)
	}
}
