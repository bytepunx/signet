package store

import (
	"context"
	"fmt"
	"time"
)

// Repository stores the configuration for a monitored git repository.
// Sensitive fields (webhook secret, deploy key) are stored encrypted.
type Repository struct {
	ID                     string
	Name                   string
	RepoURL                string
	Branch                 string
	SecretsPath            string
	ConfigPath             string
	EncryptedWebhookSecret []byte
	EncryptedDeployKey     []byte
	LastSyncSHA            string
	LastSyncAt             *time.Time
	CreatedAt              time.Time
}

// PutRepository inserts a new repository record and populates r.ID and r.CreatedAt.
// Returns ErrAlreadyExists if a repository with the same name already exists.
func (s *Store) PutRepository(ctx context.Context, r *Repository) error {
	if r.Name == "" {
		return fmt.Errorf("%w: name must not be empty", ErrInvalidInput)
	}
	if r.RepoURL == "" {
		return fmt.Errorf("%w: repo_url must not be empty", ErrInvalidInput)
	}
	if len(r.EncryptedWebhookSecret) == 0 {
		return fmt.Errorf("%w: encrypted_webhook_secret must not be empty", ErrInvalidInput)
	}
	if len(r.EncryptedDeployKey) == 0 {
		return fmt.Errorf("%w: encrypted_deploy_key must not be empty", ErrInvalidInput)
	}
	const q = `
		INSERT INTO git_repositories
			(name, repo_url, branch, secrets_path, config_path, encrypted_webhook_secret, encrypted_deploy_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	return wrapDBError("put repository",
		s.pool.QueryRow(ctx, q,
			r.Name, r.RepoURL, r.Branch, r.SecretsPath, r.ConfigPath,
			r.EncryptedWebhookSecret, r.EncryptedDeployKey,
		).Scan(&r.ID, &r.CreatedAt),
	)
}

// GetRepository returns the repository with the given ID.
// Returns ErrNotFound if absent.
func (s *Store) GetRepository(ctx context.Context, id string) (*Repository, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}
	const q = `
		SELECT id, name, repo_url, branch, secrets_path, config_path,
		       encrypted_webhook_secret, encrypted_deploy_key,
		       last_sync_sha, last_sync_at, created_at
		FROM git_repositories WHERE id = $1`
	return scanRepo(s.pool.QueryRow(ctx, q, id))
}

// GetRepositoryByName returns the repository with the given human alias.
// Returns ErrNotFound if absent.
func (s *Store) GetRepositoryByName(ctx context.Context, name string) (*Repository, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name must not be empty", ErrInvalidInput)
	}
	const q = `
		SELECT id, name, repo_url, branch, secrets_path, config_path,
		       encrypted_webhook_secret, encrypted_deploy_key,
		       last_sync_sha, last_sync_at, created_at
		FROM git_repositories WHERE name = $1`
	return scanRepo(s.pool.QueryRow(ctx, q, name))
}

// ListRepositories returns all repositories ordered by created_at ascending.
func (s *Store) ListRepositories(ctx context.Context) ([]Repository, error) {
	const q = `
		SELECT id, name, repo_url, branch, secrets_path, config_path,
		       encrypted_webhook_secret, encrypted_deploy_key,
		       last_sync_sha, last_sync_at, created_at
		FROM git_repositories
		ORDER BY created_at ASC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, wrapDBError("list repositories", err)
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		r, err := scanRepoRow(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, *r)
	}
	return repos, rows.Err()
}

// DeleteRepository permanently removes a repository record.
// Returns ErrNotFound if absent.
func (s *Store) DeleteRepository(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}
	const q = `DELETE FROM git_repositories WHERE id = $1 RETURNING id`
	var out string
	return wrapDBError("delete repository", s.pool.QueryRow(ctx, q, id).Scan(&out))
}

// UpdateSyncState records the result of a successful git sync.
func (s *Store) UpdateSyncState(ctx context.Context, id, sha string, at time.Time) error {
	if id == "" || sha == "" {
		return fmt.Errorf("%w: id and sha must not be empty", ErrInvalidInput)
	}
	const q = `
		UPDATE git_repositories
		SET last_sync_sha = $2, last_sync_at = $3
		WHERE id = $1
		RETURNING id`
	var out string
	return wrapDBError("update sync state", s.pool.QueryRow(ctx, q, id, sha, at).Scan(&out))
}

// scanRepo scans a single row returned by QueryRow into a Repository.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRepo(row rowScanner) (*Repository, error) {
	r, err := scanRepoFields(row)
	if err != nil {
		return nil, wrapDBError("scan repository", err)
	}
	return r, nil
}

func scanRepoRow(rows interface{ Scan(dest ...any) error }) (*Repository, error) {
	return scanRepoFields(rows)
}

func scanRepoFields(row rowScanner) (*Repository, error) {
	var r Repository
	err := row.Scan(
		&r.ID, &r.Name, &r.RepoURL, &r.Branch, &r.SecretsPath, &r.ConfigPath,
		&r.EncryptedWebhookSecret, &r.EncryptedDeployKey,
		&r.LastSyncSHA, &r.LastSyncAt, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
