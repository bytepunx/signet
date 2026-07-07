package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// KEK records a key-encryption-key: a random 32-byte key, itself wrapped
// (encrypted) under the master key, that in turn wraps per-secret DEKs.
// Plaintext KEK material never enters this struct.
type KEK struct {
	ID            string
	WrappedKEK    []byte
	IsActive      bool
	CreatedAt     time.Time
	DeactivatedAt *time.Time
}

// PutKEK inserts a new KEK record. The ID and CreatedAt fields are populated
// on return.
func (s *Store) PutKEK(ctx context.Context, k *KEK) error {
	if len(k.WrappedKEK) == 0 {
		return fmt.Errorf("%w: wrapped_kek must not be empty", ErrInvalidInput)
	}
	const q = `
		INSERT INTO key_encryption_keys (wrapped_kek, is_active)
		VALUES ($1, $2)
		RETURNING id, created_at`
	return wrapDBError("put kek",
		s.pool.QueryRow(ctx, q, k.WrappedKEK, k.IsActive).Scan(&k.ID, &k.CreatedAt),
	)
}

// GetActiveKEK returns the current active KEK. Returns ErrNotFound if none
// has been provisioned yet (e.g. a brand new deployment before first use).
func (s *Store) GetActiveKEK(ctx context.Context) (*KEK, error) {
	const q = `
		SELECT id, wrapped_kek, is_active, created_at, deactivated_at
		FROM key_encryption_keys
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 1`
	return scanKEK(s.pool.QueryRow(ctx, q))
}

// GetKEKByID returns the KEK with the given id, active or not. Callers need
// inactive KEKs to unwrap DEKs that predate the most recent rotation.
// Returns ErrNotFound if absent.
func (s *Store) GetKEKByID(ctx context.Context, id string) (*KEK, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}
	const q = `
		SELECT id, wrapped_kek, is_active, created_at, deactivated_at
		FROM key_encryption_keys
		WHERE id = $1`
	return scanKEK(s.pool.QueryRow(ctx, q, id))
}

// ListKEKs returns all KEKs ordered by created_at descending.
func (s *Store) ListKEKs(ctx context.Context) ([]KEK, error) {
	const q = `
		SELECT id, wrapped_kek, is_active, created_at, deactivated_at
		FROM key_encryption_keys
		ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, wrapDBError("list keks", err)
	}
	defer rows.Close()

	var keks []KEK
	for rows.Next() {
		var k KEK
		if err := rows.Scan(&k.ID, &k.WrappedKEK, &k.IsActive, &k.CreatedAt, &k.DeactivatedAt); err != nil {
			return nil, wrapDBError("scan kek", err)
		}
		keks = append(keks, k)
	}
	return keks, rows.Err()
}

// DeactivateKEK marks the KEK as inactive and records the deactivation time.
// Returns ErrNotFound if the KEK does not exist.
func (s *Store) DeactivateKEK(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}
	const q = `
		UPDATE key_encryption_keys
		SET is_active = false, deactivated_at = now()
		WHERE id = $1
		RETURNING id`
	var out string
	return wrapDBError("deactivate kek", s.pool.QueryRow(ctx, q, id).Scan(&out))
}

// CountSecretsUsingKEK returns how many secret rows currently reference the
// given KEK id. Used to guard against pruning a KEK that is still needed to
// decrypt existing secrets.
func (s *Store) CountSecretsUsingKEK(ctx context.Context, id string) (int, error) {
	if id == "" {
		return 0, fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}
	const q = `SELECT count(*) FROM secrets WHERE kek_id = $1`
	var n int
	if err := s.pool.QueryRow(ctx, q, id).Scan(&n); err != nil {
		return 0, wrapDBError("count secrets using kek", err)
	}
	return n, nil
}

// DeleteKEK permanently removes a KEK record. Callers must first verify via
// CountSecretsUsingKEK that no secret still references it, and that it is not
// the active KEK.
// Returns ErrNotFound if the KEK does not exist.
func (s *Store) DeleteKEK(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}
	const q = `DELETE FROM key_encryption_keys WHERE id = $1 RETURNING id`
	var out string
	return wrapDBError("delete kek", s.pool.QueryRow(ctx, q, id).Scan(&out))
}

func scanKEK(row pgx.Row) (*KEK, error) {
	var k KEK
	err := row.Scan(&k.ID, &k.WrappedKEK, &k.IsActive, &k.CreatedAt, &k.DeactivatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, wrapDBError("scan kek", err)
	}
	return &k, nil
}

// GetKeyCheckValue returns the stored key-check ciphertext, used to verify a
// candidate master key is correct immediately after unseal.
// Returns ErrNotFound if none has been created yet (first-ever unseal).
func (s *Store) GetKeyCheckValue(ctx context.Context) ([]byte, error) {
	const q = `SELECT ciphertext FROM key_check_value WHERE id = 'singleton'`
	var ct []byte
	err := s.pool.QueryRow(ctx, q).Scan(&ct)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, wrapDBError("get key check value", err)
	}
	return ct, nil
}

// PutKeyCheckValue creates the singleton key-check row. Returns
// ErrAlreadyExists if one has already been created — the row is immutable
// once set except via ReplaceKeyCheckValue (used during master key rotation).
func (s *Store) PutKeyCheckValue(ctx context.Context, ciphertext []byte) error {
	if len(ciphertext) == 0 {
		return fmt.Errorf("%w: ciphertext must not be empty", ErrInvalidInput)
	}
	const q = `INSERT INTO key_check_value (id, ciphertext) VALUES ('singleton', $1)`
	_, err := s.pool.Exec(ctx, q, ciphertext)
	return wrapDBError("put key check value", err)
}

// ReplaceKeyCheckValue overwrites the singleton key-check row. Used only
// during master key rotation, after the new ciphertext has been computed
// under the new master key.
func (s *Store) ReplaceKeyCheckValue(ctx context.Context, ciphertext []byte) error {
	if len(ciphertext) == 0 {
		return fmt.Errorf("%w: ciphertext must not be empty", ErrInvalidInput)
	}
	const q = `
		INSERT INTO key_check_value (id, ciphertext)
		VALUES ('singleton', $1)
		ON CONFLICT (id) DO UPDATE SET ciphertext = excluded.ciphertext, created_at = now()`
	_, err := s.pool.Exec(ctx, q, ciphertext)
	return wrapDBError("replace key check value", err)
}

// SecretKeyRef identifies a secret row's version and current DEK wrap, used
// when re-wrapping every DEK under a newly rotated KEK.
type SecretKeyRef struct {
	Namespace    string
	Service      string
	Name         string
	Version      int
	EncryptedDEK []byte
	KEKID        string // empty means the DEK is wrapped directly under the master key
}

// ListSecretKeyRefs returns the DEK wrap metadata for every secret version in
// the store. Used by KEK rotation to re-wrap each DEK under the new KEK.
func (s *Store) ListSecretKeyRefs(ctx context.Context) ([]SecretKeyRef, error) {
	const q = `SELECT namespace, service, secret_name, version, encrypted_dek, kek_id FROM secrets`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, wrapDBError("list secret key refs", err)
	}
	defer rows.Close()

	var refs []SecretKeyRef
	for rows.Next() {
		var r SecretKeyRef
		var kekID *string
		if err := rows.Scan(&r.Namespace, &r.Service, &r.Name, &r.Version, &r.EncryptedDEK, &kekID); err != nil {
			return nil, wrapDBError("scan secret key ref", err)
		}
		if kekID != nil {
			r.KEKID = *kekID
		}
		refs = append(refs, r)
	}
	return refs, rows.Err()
}

// UpdateSecretDEK rewrites the encrypted DEK and KEK reference for one exact
// secret version, without touching its ciphertext or bumping its version.
// Used by KEK rotation. Returns ErrNotFound if the row does not exist.
func (s *Store) UpdateSecretDEK(ctx context.Context, namespace, service, name string, version int, newEncDEK []byte, newKEKID string) error {
	if err := validateKey(namespace, service, name); err != nil {
		return err
	}
	if len(newEncDEK) == 0 {
		return fmt.Errorf("%w: newEncDEK must not be empty", ErrInvalidInput)
	}
	var kekID *string
	if newKEKID != "" {
		kekID = &newKEKID
	}
	const q = `
		UPDATE secrets
		SET encrypted_dek = $5, kek_id = $6, updated_at = now()
		WHERE namespace = $1 AND service = $2 AND secret_name = $3 AND version = $4
		RETURNING version`
	var out int
	err := s.pool.QueryRow(ctx, q, namespace, service, name, version, newEncDEK, kekID).Scan(&out)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return wrapDBError("update secret dek", err)
	}
	return nil
}

// KEKRewrap pairs a KEK's id with its new master-key wrapping. Used by
// RewrapKEKsAndKCV during master key rotation.
type KEKRewrap struct {
	ID         string
	WrappedKEK []byte
}

// RewrapKEKsAndKCV atomically updates every KEK's wrapped_kek and the
// key-check value's ciphertext in a single transaction. Used exclusively
// during master key rotation: kekUpdates and newKCV must already be computed
// (unwrapped under the old master key, re-wrapped under the new one) before
// calling this method. If any update fails, the whole rotation is rolled
// back and the old master key remains authoritative.
func (s *Store) RewrapKEKsAndKCV(ctx context.Context, kekUpdates []KEKRewrap, newKCV []byte) error {
	if len(newKCV) == 0 {
		return fmt.Errorf("%w: newKCV must not be empty", ErrInvalidInput)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("rewrap keks: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op if already committed

	for _, u := range kekUpdates {
		tag, err := tx.Exec(ctx,
			`UPDATE key_encryption_keys SET wrapped_kek = $2 WHERE id = $1`,
			u.ID, u.WrappedKEK,
		)
		if err != nil {
			return wrapDBError("rewrap kek", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("rewrap kek %s: %w", u.ID, ErrNotFound)
		}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO key_check_value (id, ciphertext) VALUES ('singleton', $1)
		 ON CONFLICT (id) DO UPDATE SET ciphertext = excluded.ciphertext, created_at = now()`,
		newKCV,
	); err != nil {
		return wrapDBError("rewrap kcv", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("rewrap keks: commit: %w", err)
	}
	return nil
}
