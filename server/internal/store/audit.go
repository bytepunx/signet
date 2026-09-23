package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AuditEntry records a single secret access event. Every access — permitted or
// denied — must produce an entry. HMAC chains entries for tamper detection.
type AuditEntry struct {
	// SPIFFEID is the workload identity of the caller.
	SPIFFEID string
	// Action describes the operation attempted (e.g. "get", "list", "delete").
	Action string
	// Namespace is the secret namespace targeted.
	Namespace string
	// SecretName is the name of the secret targeted.
	SecretName string
	// Outcome is "permitted" or "denied".
	Outcome string
	// PeerIP is the caller's IP address. May be empty if not available.
	PeerIP string
	// HMAC is the HMAC of this entry chained with the previous entry's HMAC.
	// Computed by internal/audit before calling WriteAuditLog.
	HMAC []byte
}

// WriteAuditLog persists an audit entry. Returns ErrInvalidInput if required
// fields are missing.
func (s *Store) WriteAuditLog(ctx context.Context, entry *AuditEntry) error {
	if err := validateAuditEntry(entry); err != nil {
		return err
	}

	const q = `
		INSERT INTO audit_log (spiffe_id, action, namespace, secret_name, outcome, peer_ip, hmac)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	var peerIP *string
	if entry.PeerIP != "" {
		peerIP = &entry.PeerIP
	}

	if _, err := s.pool.Exec(ctx, q,
		entry.SPIFFEID, entry.Action, entry.Namespace,
		entry.SecretName, entry.Outcome, peerIP, entry.HMAC,
	); err != nil {
		return wrapDBError("write audit log", err)
	}
	return nil
}

// LatestAuditHMAC returns the HMAC of the most recently written audit entry,
// used to seed the chain on startup. Returns a nil slice (not ErrNotFound) when
// the table is empty.
func (s *Store) LatestAuditHMAC(ctx context.Context) ([]byte, error) {
	var mac []byte
	err := s.pool.QueryRow(ctx,
		"SELECT hmac FROM audit_log ORDER BY ts DESC LIMIT 1",
	).Scan(&mac)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("latest audit hmac", err)
	}
	return mac, nil
}

func validateAuditEntry(e *AuditEntry) error {
	if e == nil {
		return fmt.Errorf("%w: entry must not be nil", ErrInvalidInput)
	}
	if e.SPIFFEID == "" {
		return fmt.Errorf("%w: SPIFFEID must not be empty", ErrInvalidInput)
	}
	if e.Action == "" {
		return fmt.Errorf("%w: Action must not be empty", ErrInvalidInput)
	}
	if e.Namespace == "" {
		return fmt.Errorf("%w: Namespace must not be empty", ErrInvalidInput)
	}
	if e.SecretName == "" {
		return fmt.Errorf("%w: SecretName must not be empty", ErrInvalidInput)
	}
	if e.Outcome == "" {
		return fmt.Errorf("%w: Outcome must not be empty", ErrInvalidInput)
	}
	if len(e.HMAC) == 0 {
		return fmt.Errorf("%w: HMAC must not be empty", ErrInvalidInput)
	}
	return nil
}
