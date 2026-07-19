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
	// KEKID identifies the key-encryption-key that wraps EncryptedDEK. Empty
	// means EncryptedDEK is wrapped directly under the master key — the shape
	// used before the KEK tier was introduced; decryption code must handle
	// both forms.
	KEKID     string
	ExpiresAt *time.Time
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
	// RepoID identifies the git repository (see git_repositories) this
	// version was synced from, if any. Empty for secrets written outside a
	// registered repo sync (e.g. "signet bundle push", or rows written
	// before this field existed). Used by FullSync to detect secrets that
	// have been removed from a repo since the last sync — see
	// ListSecretKeysForRepo.
	RepoID string
}

// SecretKey identifies a secret without any of its content, for the
// existence-diffing FullSync uses to detect deletions.
type SecretKey struct {
	Namespace string
	Service   string
	Name      string
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

	var kekID *string
	if sec.KEKID != "" {
		kekID = &sec.KEKID
	}
	var repoID *string
	if sec.RepoID != "" {
		repoID = &sec.RepoID
	}

	// Auto-increment version atomically using a CTE.
	const q = `
		WITH next_v AS (
			SELECT COALESCE(MAX(version), 0) + 1 AS v
			FROM secrets
			WHERE namespace = $1 AND service = $2 AND secret_name = $3
		)
		INSERT INTO secrets
			(namespace, service, secret_name, version, encrypted_dek, ciphertext, expires_at, metadata, kek_id, repo_id)
		SELECT $1, $2, $3, next_v.v, $4, $5, $6, $7, $8, $9
		FROM next_v
		RETURNING version, created_at, updated_at`

	err := s.pool.QueryRow(ctx, q,
		sec.Namespace, sec.Service, sec.Name,
		sec.EncryptedDEK, sec.Ciphertext,
		sec.ExpiresAt, metadata, kekID, repoID,
	).Scan(&sec.Version, &sec.CreatedAt, &sec.UpdatedAt)
	if err != nil {
		return wrapDBError("put secret", err)
	}
	return nil
}

// UpdateSecretRepoID sets repo_id on the latest version of the named secret
// in place, without creating a new version or touching the ciphertext.
//
// storeSecret's dedup optimization (see isUnchanged in internal/gitops)
// skips PutSecret entirely when a resync's plaintext matches what's already
// stored, to bound version growth across repeated reconciliation passes —
// but that means a secret whose content simply hasn't changed since before
// repo_id existed, or since it was last synced by a different repo, would
// otherwise never pick up the current repoID and so could never become a
// deletion-detection candidate (see ListSecretKeysForRepo). This call is
// how the dedup path still keeps attribution current.
func (s *Store) UpdateSecretRepoID(ctx context.Context, namespace, service, name, repoID string) error {
	if err := validateKey(namespace, service, name); err != nil {
		return err
	}
	if repoID == "" {
		return fmt.Errorf("%w: repoID must not be empty", ErrInvalidInput)
	}
	const q = `
		UPDATE secrets SET repo_id = $4
		WHERE namespace = $1 AND service = $2 AND secret_name = $3
		  AND version = (
		      SELECT MAX(version) FROM secrets
		      WHERE namespace = $1 AND service = $2 AND secret_name = $3
		  )`
	tag, err := s.pool.Exec(ctx, q, namespace, service, name, repoID)
	if err != nil {
		return wrapDBError("update secret repo id", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSecret returns the latest non-expired version of the named secret. A
// version whose expires_at is in the past is skipped in favor of the next
// most recent non-expired version, if any. Returns ErrNotFound if no such
// secret exists, or if every version has expired — an expired secret is
// treated as absent rather than served past its intended lifetime.
func (s *Store) GetSecret(ctx context.Context, namespace, service, name string) (*Secret, error) {
	if err := validateKey(namespace, service, name); err != nil {
		return nil, err
	}

	const q = `
		SELECT namespace, service, secret_name, version,
		       encrypted_dek, ciphertext, expires_at, metadata,
		       created_at, updated_at, kek_id
		FROM secrets
		WHERE namespace = $1 AND service = $2 AND secret_name = $3
		  AND (expires_at IS NULL OR expires_at > now())
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
		       created_at, updated_at, kek_id
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

// FetchServiceSecrets returns the latest non-expired version of every secret
// stored under (namespace, service), including encrypted payload, for bundle
// decryption. A secret whose latest version has expired is skipped in favor
// of its latest non-expired version, if any; if every version has expired it
// is omitted entirely rather than returned past its intended lifetime.
// Returns an empty slice (not ErrNotFound) when no secrets exist.
func (s *Store) FetchServiceSecrets(ctx context.Context, namespace, service string) ([]Secret, error) {
	if namespace == "" || service == "" {
		return nil, fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidInput)
	}
	const q = `
		SELECT DISTINCT ON (secret_name)
		       namespace, service, secret_name, version,
		       encrypted_dek, ciphertext, expires_at, metadata,
		       created_at, updated_at, kek_id
		FROM secrets
		WHERE namespace = $1 AND service = $2
		  AND (expires_at IS NULL OR expires_at > now())
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
			kekID    *string
		)
		if err := rows.Scan(
			&sec.Namespace, &sec.Service, &sec.Name, &sec.Version,
			&sec.EncryptedDEK, &sec.Ciphertext,
			&sec.ExpiresAt, &metadata,
			&sec.CreatedAt, &sec.UpdatedAt, &kekID,
		); err != nil {
			return nil, fmt.Errorf("fetch service secrets: scan row: %w", err)
		}
		if kekID != nil {
			sec.KEKID = *kekID
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

// ListSecretKeysForRepo returns the (namespace, service, name) of every
// secret whose latest version is currently attributed to repoID (see
// Secret.RepoID). Used by FullSync to compute which secrets it previously
// synced from this repo are no longer present in it.
func (s *Store) ListSecretKeysForRepo(ctx context.Context, repoID string) ([]SecretKey, error) {
	if repoID == "" {
		return nil, fmt.Errorf("%w: repoID must not be empty", ErrInvalidInput)
	}
	const q = `
		SELECT namespace, service, secret_name
		FROM (
			SELECT DISTINCT ON (namespace, service, secret_name)
			       namespace, service, secret_name, repo_id
			FROM secrets
			ORDER BY namespace, service, secret_name, version DESC
		) latest
		WHERE repo_id = $1`
	rows, err := s.pool.Query(ctx, q, repoID)
	if err != nil {
		return nil, wrapDBError("list secret keys for repo", err)
	}
	defer rows.Close()

	var keys []SecretKey
	for rows.Next() {
		var k SecretKey
		if err := rows.Scan(&k.Namespace, &k.Service, &k.Name); err != nil {
			return nil, fmt.Errorf("list secret keys for repo: scan row: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBError("list secret keys for repo", err)
	}
	return keys, nil
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
		kekID    *string
	)
	err := row.Scan(
		&sec.Namespace, &sec.Service, &sec.Name, &sec.Version,
		&sec.EncryptedDEK, &sec.Ciphertext,
		&sec.ExpiresAt, &metadata,
		&sec.CreatedAt, &sec.UpdatedAt, &kekID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, wrapDBError("scan secret", err)
	}
	if kekID != nil {
		sec.KEKID = *kekID
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
