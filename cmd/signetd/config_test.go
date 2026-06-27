package main

import (
	"encoding/hex"
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
