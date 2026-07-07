package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// config holds all runtime configuration for signetd.
// Values are loaded from CLI flags first; if a flag is not provided, the
// corresponding SIGNET_* environment variable is used; finally, a built-in
// default applies.
//
// Key material (master key, DEKs) never appears in config. The audit chain key
// is the only secret here; it is read from SIGNET_AUDIT_CHAIN_KEY (inline hex)
// or SIGNET_AUDIT_CHAIN_KEY_FILE (path to a file containing the hex), exactly
// one of which must be set. A file (e.g. a projected Kubernetes Secret volume)
// avoids the key sitting in process environment, which is readable via
// /proc/<pid>/environ and sometimes captured in crash dumps or telemetry.
type config struct {
	// Database
	DBConnString string // SIGNET_DB_CONN_STRING

	// SPIRE / identity
	SpireSocket string // SIGNET_SPIRE_SOCKET
	TrustDomain string // SIGNET_TRUST_DOMAIN

	// Listener addresses
	WorkloadAddr   string        // SIGNET_WORKLOAD_ADDR
	AdminAddr      string        // SIGNET_ADMIN_ADDR
	WebhookAddr    string        // SIGNET_WEBHOOK_ADDR
	WebhookBaseURL string        // SIGNET_WEBHOOK_BASE_URL
	DrainTimeout   time.Duration // SIGNET_DRAIN_TIMEOUT

	// Shamir unseal (both zero → direct key mode)
	ShamirShares    int           // SIGNET_SHAMIR_SHARES
	ShamirThreshold int           // SIGNET_SHAMIR_THRESHOLD
	ShareTimeout    time.Duration // SIGNET_SHARE_TIMEOUT

	// Kubernetes SA token validation
	kubeAudiences []string // parsed from SIGNET_KUBE_AUDIENCES (comma-separated); must not end up empty
	adminSubjects []string // parsed from SIGNET_ADMIN_SUBJECTS (comma-separated); may be empty if relying solely on RBAC

	// Kubernetes auto-unseal (optional — empty means disabled)
	KubeUnsealSecret string // SIGNET_KUBE_UNSEAL_SECRET

	// Environment label for this signet instance (optional).
	// When set, only SOPS age keys tagged for this environment (or global keys)
	// are loaded for decryption. Conventionally: "prod", "staging", "dev".
	Environment string // SIGNET_ENVIRONMENT

	// Audit HMAC chain key (hex-encoded, must decode to exactly 32 bytes)
	auditKeyBytes []byte // decoded from SIGNET_AUDIT_CHAIN_KEY

	// AuditFailClosed, when true (the default), denies an otherwise-permitted
	// secret/config/bundle access if its audit log write fails, rather than
	// serving it without a durable audit trail.
	AuditFailClosed bool // SIGNET_AUDIT_FAIL_CLOSED
}

// loadConfig registers CLI flags, parses them, then validates the result.
// Returns a detailed error if any required field is missing or invalid.
func loadConfig() (config, error) {
	var raw struct {
		dbConnString      string
		spireSocket       string
		trustDomain       string
		workloadAddr      string
		adminAddr         string
		webhookAddr       string
		webhookBaseURL    string
		drainTimeout      string
		shamirShares      int
		shamirThreshold   int
		shareTimeout      string
		kubeAudiences     string
		adminSubjects     string
		kubeUnsealSecret  string
		environment       string
		auditChainKey     string
		auditChainKeyFile string
		auditFailClosed   bool
	}

	flag.StringVar(&raw.dbConnString, "db", envOr("SIGNET_DB_CONN_STRING", ""),
		"CockroachDB connection string (required; env: SIGNET_DB_CONN_STRING)")
	flag.StringVar(&raw.spireSocket, "spire-socket", envOr("SIGNET_SPIRE_SOCKET", "unix:///run/spire/sockets/agent.sock"),
		"SPIRE workload API socket path (env: SIGNET_SPIRE_SOCKET)")
	flag.StringVar(&raw.trustDomain, "trust-domain", envOr("SIGNET_TRUST_DOMAIN", ""),
		"SPIFFE trust domain, e.g. example.org (required; env: SIGNET_TRUST_DOMAIN)")
	flag.StringVar(&raw.workloadAddr, "workload-addr", envOr("SIGNET_WORKLOAD_ADDR", ":8443"),
		"workload mTLS gRPC listener address (env: SIGNET_WORKLOAD_ADDR)")
	flag.StringVar(&raw.adminAddr, "admin-addr", envOr("SIGNET_ADMIN_ADDR", "127.0.0.1:8444"),
		"admin gRPC listener address — expose only via kubectl port-forward (env: SIGNET_ADMIN_ADDR)")
	flag.StringVar(&raw.webhookAddr, "webhook-addr", envOr("SIGNET_WEBHOOK_ADDR", ":8445"),
		"GitHub webhook HTTP listener address (env: SIGNET_WEBHOOK_ADDR); empty to disable")
	flag.StringVar(&raw.webhookBaseURL, "webhook-base-url", envOr("SIGNET_WEBHOOK_BASE_URL", ""),
		"public base URL for webhook callbacks, e.g. https://signet.example.com (env: SIGNET_WEBHOOK_BASE_URL)")
	flag.StringVar(&raw.drainTimeout, "drain-timeout", envOr("SIGNET_DRAIN_TIMEOUT", "30s"),
		"graceful shutdown drain period, e.g. 30s (env: SIGNET_DRAIN_TIMEOUT)")
	flag.IntVar(&raw.shamirShares, "shamir-shares", envOrInt("SIGNET_SHAMIR_SHARES", 0),
		"total Shamir shares (n); 0 = direct key mode (env: SIGNET_SHAMIR_SHARES)")
	flag.IntVar(&raw.shamirThreshold, "shamir-threshold", envOrInt("SIGNET_SHAMIR_THRESHOLD", 0),
		"Shamir threshold (t); must satisfy 2 ≤ t ≤ n (env: SIGNET_SHAMIR_THRESHOLD)")
	flag.StringVar(&raw.shareTimeout, "share-timeout", envOr("SIGNET_SHARE_TIMEOUT", "30m"),
		"how long to wait for all Shamir shares before expiring (env: SIGNET_SHARE_TIMEOUT)")
	flag.StringVar(&raw.kubeAudiences, "kube-audiences", envOr("SIGNET_KUBE_AUDIENCES", "signet"),
		"comma-separated Kubernetes SA token audiences for admin auth; must not be empty (env: SIGNET_KUBE_AUDIENCES)")
	flag.StringVar(&raw.adminSubjects, "admin-subjects", envOr("SIGNET_ADMIN_SUBJECTS", ""),
		"comma-separated allowlist of admin identities, each 'serviceaccount:<namespace>:<name>' or "+
			"'group:<name>'; identities not listed here can still be authorized via RBAC on "+
			"signet.io/adminoperations verb administer (env: SIGNET_ADMIN_SUBJECTS)")
	flag.StringVar(&raw.kubeUnsealSecret, "kube-unseal-secret", envOr("SIGNET_KUBE_UNSEAL_SECRET", ""),
		"name of the Kubernetes Secret holding the master key for auto-unseal on startup; empty disables (env: SIGNET_KUBE_UNSEAL_SECRET)")
	flag.StringVar(&raw.environment, "environment", envOr("SIGNET_ENVIRONMENT", ""),
		"environment label for this instance, e.g. prod, staging, dev; scopes SOPS key generation and decryption (env: SIGNET_ENVIRONMENT)")
	flag.StringVar(&raw.auditChainKey, "audit-chain-key", envOr("SIGNET_AUDIT_CHAIN_KEY", ""),
		"64-character hex-encoded 32-byte HMAC chain key for audit log integrity; "+
			"mutually exclusive with -audit-chain-key-file, exactly one is required (env: SIGNET_AUDIT_CHAIN_KEY)")
	flag.StringVar(&raw.auditChainKeyFile, "audit-chain-key-file", envOr("SIGNET_AUDIT_CHAIN_KEY_FILE", ""),
		"path to a file containing the hex-encoded audit chain key, as an alternative to passing it "+
			"inline via -audit-chain-key (avoids the key sitting in process environment) (env: SIGNET_AUDIT_CHAIN_KEY_FILE)")
	flag.BoolVar(&raw.auditFailClosed, "audit-fail-closed", envOrBool("SIGNET_AUDIT_FAIL_CLOSED", true),
		"deny an otherwise-permitted secret/config/bundle access if its audit log write fails (recommended; env: SIGNET_AUDIT_FAIL_CLOSED)")

	flag.Parse()
	return validate(raw)
}

type rawConfig = struct {
	dbConnString      string
	spireSocket       string
	trustDomain       string
	workloadAddr      string
	adminAddr         string
	webhookAddr       string
	webhookBaseURL    string
	drainTimeout      string
	shamirShares      int
	shamirThreshold   int
	shareTimeout      string
	kubeAudiences     string
	adminSubjects     string
	kubeUnsealSecret  string
	environment       string
	auditChainKey     string
	auditChainKeyFile string
	auditFailClosed   bool
}

func validate(raw rawConfig) (config, error) {
	var errs []string
	require := func(v, name string) {
		if v == "" {
			errs = append(errs, fmt.Sprintf("%s is required", name))
		}
	}

	require(raw.dbConnString, "-db / SIGNET_DB_CONN_STRING")
	require(raw.trustDomain, "-trust-domain / SIGNET_TRUST_DOMAIN")

	auditChainKeyHex := raw.auditChainKey
	switch {
	case raw.auditChainKey != "" && raw.auditChainKeyFile != "":
		errs = append(errs, "specify exactly one of -audit-chain-key / SIGNET_AUDIT_CHAIN_KEY or "+
			"-audit-chain-key-file / SIGNET_AUDIT_CHAIN_KEY_FILE, not both")
	case raw.auditChainKeyFile != "":
		data, readErr := os.ReadFile(raw.auditChainKeyFile)
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("read -audit-chain-key-file %s: %v", raw.auditChainKeyFile, readErr))
		} else {
			auditChainKeyHex = strings.TrimSpace(string(data))
		}
	case raw.auditChainKey == "":
		errs = append(errs, "one of -audit-chain-key / SIGNET_AUDIT_CHAIN_KEY or "+
			"-audit-chain-key-file / SIGNET_AUDIT_CHAIN_KEY_FILE is required")
	}

	drainTimeout, err := time.ParseDuration(raw.drainTimeout)
	if err != nil {
		errs = append(errs, fmt.Sprintf("invalid -drain-timeout %q: %v", raw.drainTimeout, err))
	}
	shareTimeout, err := time.ParseDuration(raw.shareTimeout)
	if err != nil {
		errs = append(errs, fmt.Sprintf("invalid -share-timeout %q: %v", raw.shareTimeout, err))
	}

	var auditKey []byte
	if auditChainKeyHex != "" {
		key, decErr := hex.DecodeString(auditChainKeyHex)
		switch {
		case decErr != nil:
			errs = append(errs, fmt.Sprintf("invalid audit chain key: not valid hex: %v", decErr))
		case len(key) != 32:
			errs = append(errs, fmt.Sprintf("invalid audit chain key: must be 32 bytes (64 hex chars), got %d bytes", len(key)))
		default:
			auditKey = key
		}
	}

	if raw.shamirThreshold > 0 || raw.shamirShares > 0 {
		if raw.shamirThreshold < 2 {
			errs = append(errs, fmt.Sprintf("shamir-threshold must be ≥ 2, got %d", raw.shamirThreshold))
		}
		if raw.shamirShares < raw.shamirThreshold {
			errs = append(errs, fmt.Sprintf("shamir-shares (%d) must be ≥ shamir-threshold (%d)", raw.shamirShares, raw.shamirThreshold))
		}
	}

	audiences := splitNonEmpty(raw.kubeAudiences)
	if len(audiences) == 0 {
		// An empty audience list makes the underlying Kubernetes TokenReview
		// accept a token bearing ANY audience, including the default
		// "https://kubernetes.default.svc" audience mounted into virtually
		// every pod. Refuse to start rather than silently widen who can even
		// attempt admin authentication.
		errs = append(errs, "-kube-audiences / SIGNET_KUBE_AUDIENCES must not be empty")
	}

	if len(errs) > 0 {
		return config{}, fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return config{
		DBConnString:     raw.dbConnString,
		SpireSocket:      raw.spireSocket,
		TrustDomain:      raw.trustDomain,
		WorkloadAddr:     raw.workloadAddr,
		AdminAddr:        raw.adminAddr,
		WebhookAddr:      raw.webhookAddr,
		WebhookBaseURL:   raw.webhookBaseURL,
		DrainTimeout:     drainTimeout,
		ShamirShares:     raw.shamirShares,
		ShamirThreshold:  raw.shamirThreshold,
		ShareTimeout:     shareTimeout,
		kubeAudiences:    audiences,
		adminSubjects:    splitNonEmpty(raw.adminSubjects),
		KubeUnsealSecret: raw.kubeUnsealSecret,
		Environment:      raw.environment,
		auditKeyBytes:    auditKey,
		AuditFailClosed:  raw.auditFailClosed,
	}, nil
}

// splitNonEmpty splits a comma-separated string, trimming whitespace and
// dropping empty entries. Returns nil (not an empty slice) if nothing remains.
func splitNonEmpty(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscan(v, &n); err != nil {
		return fallback
	}
	return n
}

func envOrBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
