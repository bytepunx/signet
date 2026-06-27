package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Secret holds encrypted secret material. EncryptedDEK and Ciphertext are the
// only payload fields — plaintext never passes through this layer.
type Secret struct {
	Namespace    string
	Service      string
	Name         string
	Version      int
	EncryptedDEK []byte
	Ciphertext   []byte
	ExpiresAt    *time.Time
	Metadata     map[string]string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SecretMeta describes a secret without its encrypted payload. Used for listing.
type SecretMeta struct {
	Namespace string
	Service   string
	Name      string
	Version   int
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PutSecret inserts a new version of the secret. The version is auto-incremented
// from the current maximum for that (namespace, service, name) tuple. The Version,
// CreatedAt, and UpdatedAt fields of s are populated on return.
func (s *Store) PutSecret(ctx context.Context, sec *Secret) error {
	if err := validateSecret(sec); err != nil {
		return err
	}

	var metadata []byte
	if len(sec.Metadata) > 0 {
		var err error
		metadata, err = json.Marshal(sec.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	// Auto-increment version atomically using a CTE.
	const q = `
		WITH next_v AS (
			SELECT COALESCE(MAX(version), 0) + 1 AS v
			FROM secrets
			WHERE namespace = $1 AND service = $2 AND secret_name = $3
		)
		INSERT INTO secrets
			(namespace, service, secret_name, version, encrypted_dek, ciphertext, expires_at, metadata)
		SELECT $1, $2, $3, next_v.v, $4, $5, $6, $7
		FROM next_v
		RETURNING version, created_at, updated_at`

	err := s.pool.QueryRow(ctx, q,
		sec.Namespace, sec.Service, sec.Name,
		sec.EncryptedDEK, sec.Ciphertext,
		sec.ExpiresAt, metadata,
	).Scan(&sec.Version, &sec.CreatedAt, &sec.UpdatedAt)
	if err != nil {
		return wrapDBError("put secret", err)
	}
	return nil
}

// GetSecret returns the latest version of the named secret.
// Returns ErrNotFound if no such secret exists.
func (s *Store) GetSecret(ctx context.Context, namespace, service, name string) (*Secret, error) {
	if err := validateKey(namespace, service, name); err != nil {
		return nil, err
	}

	const q = `
		SELECT namespace, service, secret_name, version,
		       encrypted_dek, ciphertext, expires_at, metadata,
		       created_at, updated_at
		FROM secrets
		WHERE namespace = $1 AND service = $2 AND secret_name = $3
		ORDER BY version DESC
		LIMIT 1`

	return scanSecret(s.pool.QueryRow(ctx, q, namespace, service, name))
}

// GetSecretAtVersion returns the specific version of the named secret.
// Returns ErrNotFound if that version does not exist.
func (s *Store) GetSecretAtVersion(ctx context.Context, namespace, service, name string, version int) (*Secret, error) {
	if err := validateKey(namespace, service, name); err != nil {
		return nil, err
	}
	if version < 1 {
		return nil, fmt.Errorf("%w: version must be >= 1", ErrInvalidInput)
	}

	const q = `
		SELECT namespace, service, secret_name, version,
		       encrypted_dek, ciphertext, expires_at, metadata,
		       created_at, updated_at
		FROM secrets
		WHERE namespace = $1 AND service = $2 AND secret_name = $3 AND version = $4`

	return scanSecret(s.pool.QueryRow(ctx, q, namespace, service, name, version))
}

// ListSecrets returns metadata for all secrets under the given namespace and service.
// Pass an empty service to list across all services in the namespace.
func (s *Store) ListSecrets(ctx context.Context, namespace, service string) ([]SecretMeta, error) {
	if namespace == "" {
		return nil, fmt.Errorf("%w: namespace must not be empty", ErrInvalidInput)
	}

	var (
		rows pgx.Rows
		err  error
	)
	if service == "" {
		const q = `
			SELECT DISTINCT ON (namespace, service, secret_name)
			       namespace, service, secret_name, version, expires_at, created_at, updated_at
			FROM secrets
			WHERE namespace = $1
			ORDER BY namespace, service, secret_name, version DESC`
		rows, err = s.pool.Query(ctx, q, namespace)
	} else {
		const q = `
			SELECT DISTINCT ON (namespace, service, secret_name)
			       namespace, service, secret_name, version, expires_at, created_at, updated_at
			FROM secrets
			WHERE namespace = $1 AND service = $2
			ORDER BY namespace, service, secret_name, version DESC`
		rows, err = s.pool.Query(ctx, q, namespace, service)
	}
	if err != nil {
		return nil, wrapDBError("list secrets", err)
	}
	defer rows.Close()

	var metas []SecretMeta
	for rows.Next() {
		var m SecretMeta
		if err := rows.Scan(
			&m.Namespace, &m.Service, &m.Name, &m.Version,
			&m.ExpiresAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list secrets: scan row: %w", err)
		}
		metas = append(metas, m)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBError("list secrets", err)
	}
	return metas, nil
}

// FetchServiceSecrets returns the latest version of every secret stored under
// (namespace, service), including encrypted payload, for bundle decryption.
// Returns an empty slice (not ErrNotFound) when no secrets exist.
func (s *Store) FetchServiceSecrets(ctx context.Context, namespace, service string) ([]Secret, error) {
	if namespace == "" || service == "" {
		return nil, fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidInput)
	}
	const q = `
		SELECT DISTINCT ON (secret_name)
		       namespace, service, secret_name, version,
		       encrypted_dek, ciphertext, expires_at, metadata,
		       created_at, updated_at
		FROM secrets
		WHERE namespace = $1 AND service = $2
		ORDER BY secret_name, version DESC`
	rows, err := s.pool.Query(ctx, q, namespace, service)
	if err != nil {
		return nil, wrapDBError("fetch service secrets", err)
	}
	defer rows.Close()

	var secrets []Secret
	for rows.Next() {
		var (
			sec      Secret
			metadata []byte
		)
		if err := rows.Scan(
			&sec.Namespace, &sec.Service, &sec.Name, &sec.Version,
			&sec.EncryptedDEK, &sec.Ciphertext,
			&sec.ExpiresAt, &metadata,
			&sec.CreatedAt, &sec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("fetch service secrets: scan row: %w", err)
		}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &sec.Metadata); err != nil {
				return nil, fmt.Errorf("fetch service secrets: unmarshal metadata: %w", err)
			}
		}
		secrets = append(secrets, sec)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBError("fetch service secrets", err)
	}
	return secrets, nil
}

// DeleteSecret removes all versions of the named secret.
// Returns ErrNotFound if no such secret exists.
func (s *Store) DeleteSecret(ctx context.Context, namespace, service, name string) error {
	if err := validateKey(namespace, service, name); err != nil {
		return err
	}

	const q = `
		DELETE FROM secrets
		WHERE namespace = $1 AND service = $2 AND secret_name = $3`

	tag, err := s.pool.Exec(ctx, q, namespace, service, name)
	if err != nil {
		return wrapDBError("delete secret", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- helpers ---

func validateSecret(sec *Secret) error {
	if sec == nil {
		return fmt.Errorf("%w: secret must not be nil", ErrInvalidInput)
	}
	if err := validateKey(sec.Namespace, sec.Service, sec.Name); err != nil {
		return err
	}
	if len(sec.EncryptedDEK) == 0 {
		return fmt.Errorf("%w: EncryptedDEK must not be empty", ErrInvalidInput)
	}
	if len(sec.Ciphertext) == 0 {
		return fmt.Errorf("%w: Ciphertext must not be empty", ErrInvalidInput)
	}
	return nil
}

func validateKey(namespace, service, name string) error {
	if namespace == "" {
		return fmt.Errorf("%w: namespace must not be empty", ErrInvalidInput)
	}
	if service == "" {
		return fmt.Errorf("%w: service must not be empty", ErrInvalidInput)
	}
	if name == "" {
		return fmt.Errorf("%w: name must not be empty", ErrInvalidInput)
	}
	return nil
}

func scanSecret(row pgx.Row) (*Secret, error) {
	var (
		sec      Secret
		metadata []byte
	)
	err := row.Scan(
		&sec.Namespace, &sec.Service, &sec.Name, &sec.Version,
		&sec.EncryptedDEK, &sec.Ciphertext,
		&sec.ExpiresAt, &metadata,
		&sec.CreatedAt, &sec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, wrapDBError("scan secret", err)
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &sec.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return &sec, nil
}

// wrapDBError maps pgx/pgconn errors to typed store errors where possible.
func wrapDBError(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", ErrAlreadyExists, pgErr.Detail)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: foreign key constraint: %s", ErrInvalidInput, pgErr.Detail)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
