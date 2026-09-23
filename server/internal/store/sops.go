package store

import (
	"context"
	"fmt"
	"time"
)

// SOPSKey records an age keypair managed by signet.
// The private key is always stored encrypted; plaintext private key material
// never enters this struct.
//
// Environment scoping:
//   - Environment == "" → global key, usable by any signet instance
//   - Environment != "" → scoped to that environment (e.g. "prod", "staging")
//
// When a signet instance loads identities for decryption it returns keys where
// Environment matches the instance's own SIGNET_ENVIRONMENT, plus all global
// (empty) keys. If the instance has no environment configured it returns all keys.
type SOPSKey struct {
	PublicKey           string
	EncryptedPrivateKey []byte
	Environment         string
	IsActive            bool
	CreatedAt           time.Time
	DeactivatedAt       *time.Time
}

// PutSOPSKey inserts a new age key record. Returns ErrAlreadyExists if the
// public key is already present.
func (s *Store) PutSOPSKey(ctx context.Context, key *SOPSKey) error {
	if key.PublicKey == "" {
		return fmt.Errorf("%w: public_key must not be empty", ErrInvalidInput)
	}
	if len(key.EncryptedPrivateKey) == 0 {
		return fmt.Errorf("%w: encrypted_private_key must not be empty", ErrInvalidInput)
	}
	const q = `
		INSERT INTO sops_age_keys (public_key, encrypted_private_key, environment, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`
	return wrapDBError("put sops key",
		s.pool.QueryRow(ctx, q,
			key.PublicKey, key.EncryptedPrivateKey, key.Environment, key.IsActive,
		).Scan(&key.CreatedAt),
	)
}

// GetSOPSKey retrieves the key with the given public key.
// Returns ErrNotFound if absent.
func (s *Store) GetSOPSKey(ctx context.Context, pubKey string) (*SOPSKey, error) {
	if pubKey == "" {
		return nil, fmt.Errorf("%w: public_key must not be empty", ErrInvalidInput)
	}
	const q = `
		SELECT public_key, encrypted_private_key, environment, is_active, created_at, deactivated_at
		FROM sops_age_keys
		WHERE public_key = $1`
	var k SOPSKey
	err := s.pool.QueryRow(ctx, q, pubKey).Scan(
		&k.PublicKey, &k.EncryptedPrivateKey, &k.Environment, &k.IsActive, &k.CreatedAt, &k.DeactivatedAt,
	)
	if err != nil {
		return nil, wrapDBError("get sops key", err)
	}
	return &k, nil
}

// GetActiveSOPSKey returns the most recently created active age key that is
// visible to the given environment. When env is empty all active keys are
// considered; otherwise only keys with a matching environment or a global
// (empty) environment are returned.
//
// Returns ErrNotFound if no matching active key exists.
func (s *Store) GetActiveSOPSKey(ctx context.Context, env string) (*SOPSKey, error) {
	var q string
	var args []any
	if env == "" {
		q = `
			SELECT public_key, encrypted_private_key, environment, is_active, created_at, deactivated_at
			FROM sops_age_keys
			WHERE is_active = true
			ORDER BY created_at DESC
			LIMIT 1`
	} else {
		q = `
			SELECT public_key, encrypted_private_key, environment, is_active, created_at, deactivated_at
			FROM sops_age_keys
			WHERE is_active = true AND (environment = $1 OR environment = '')
			ORDER BY environment DESC, created_at DESC
			LIMIT 1`
		args = []any{env}
	}
	var k SOPSKey
	err := s.pool.QueryRow(ctx, q, args...).Scan(
		&k.PublicKey, &k.EncryptedPrivateKey, &k.Environment, &k.IsActive, &k.CreatedAt, &k.DeactivatedAt,
	)
	if err != nil {
		return nil, wrapDBError("get active sops key", err)
	}
	return &k, nil
}

// ListSOPSKeys returns age keys ordered by created_at descending.
// When env is empty, all keys are returned (unfiltered).
// When env is non-empty, only keys with a matching environment or a global
// (empty) environment are returned.
func (s *Store) ListSOPSKeys(ctx context.Context, env string) ([]SOPSKey, error) {
	var q string
	var args []any
	if env == "" {
		q = `
			SELECT public_key, encrypted_private_key, environment, is_active, created_at, deactivated_at
			FROM sops_age_keys
			ORDER BY created_at DESC`
	} else {
		q = `
			SELECT public_key, encrypted_private_key, environment, is_active, created_at, deactivated_at
			FROM sops_age_keys
			WHERE environment = $1 OR environment = ''
			ORDER BY created_at DESC`
		args = []any{env}
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, wrapDBError("list sops keys", err)
	}
	defer rows.Close()

	var keys []SOPSKey
	for rows.Next() {
		var k SOPSKey
		if err := rows.Scan(
			&k.PublicKey, &k.EncryptedPrivateKey, &k.Environment, &k.IsActive, &k.CreatedAt, &k.DeactivatedAt,
		); err != nil {
			return nil, wrapDBError("scan sops key", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeactivateSOPSKey marks the key as inactive and records the deactivation time.
// Returns ErrNotFound if the key does not exist.
func (s *Store) DeactivateSOPSKey(ctx context.Context, pubKey string) error {
	if pubKey == "" {
		return fmt.Errorf("%w: public_key must not be empty", ErrInvalidInput)
	}
	const q = `
		UPDATE sops_age_keys
		SET is_active = false, deactivated_at = now()
		WHERE public_key = $1
		RETURNING public_key`
	var out string
	err := s.pool.QueryRow(ctx, q, pubKey).Scan(&out)
	return wrapDBError("deactivate sops key", err)
}

// DeleteSOPSKey permanently removes an age key.
// Returns ErrNotFound if the key does not exist.
func (s *Store) DeleteSOPSKey(ctx context.Context, pubKey string) error {
	if pubKey == "" {
		return fmt.Errorf("%w: public_key must not be empty", ErrInvalidInput)
	}
	const q = `DELETE FROM sops_age_keys WHERE public_key = $1 RETURNING public_key`
	var out string
	err := s.pool.QueryRow(ctx, q, pubKey).Scan(&out)
	return wrapDBError("delete sops key", err)
}
