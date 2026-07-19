package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// PutServiceConfig replaces the full configuration document for (namespace, service).
// content must be a valid JSON object marshaled from the service's config YAML.
// The version is incremented on each update. repoID identifies the git
// repository (see git_repositories) this config was synced from, if any —
// pass "" for configs written outside a registered repo sync (e.g. "signet
// bundle push"). See ListConfigKeysForRepo.
func (s *Store) PutServiceConfig(ctx context.Context, namespace, service string, content json.RawMessage, repoID string) error {
	if namespace == "" || service == "" {
		return fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidInput)
	}
	if len(content) == 0 {
		return fmt.Errorf("%w: content must not be empty", ErrInvalidInput)
	}
	var repoIDArg *string
	if repoID != "" {
		repoIDArg = &repoID
	}
	const q = `
		INSERT INTO configs (namespace, service, content, version, repo_id)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (namespace, service) DO UPDATE
			SET content    = excluded.content,
			    version    = configs.version + 1,
			    updated_at = now(),
			    repo_id    = excluded.repo_id`
	_, err := s.pool.Exec(ctx, q, namespace, service, content, repoIDArg)
	return wrapDBError("put service config", err)
}

// ConfigKey identifies a service config without its content, for the
// existence-diffing FullSync uses to detect deletions.
type ConfigKey struct {
	Namespace string
	Service   string
}

// ListConfigKeysForRepo returns the (namespace, service) of every config
// currently attributed to repoID (see PutServiceConfig's repoID parameter).
// Used by FullSync to compute which configs it previously synced from this
// repo are no longer present in it.
func (s *Store) ListConfigKeysForRepo(ctx context.Context, repoID string) ([]ConfigKey, error) {
	if repoID == "" {
		return nil, fmt.Errorf("%w: repoID must not be empty", ErrInvalidInput)
	}
	const q = `SELECT namespace, service FROM configs WHERE repo_id = $1`
	rows, err := s.pool.Query(ctx, q, repoID)
	if err != nil {
		return nil, wrapDBError("list config keys for repo", err)
	}
	defer rows.Close()

	var keys []ConfigKey
	for rows.Next() {
		var k ConfigKey
		if err := rows.Scan(&k.Namespace, &k.Service); err != nil {
			return nil, fmt.Errorf("list config keys for repo: scan row: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBError("list config keys for repo", err)
	}
	return keys, nil
}

// GetServiceConfig returns the full JSON configuration document and version for
// (namespace, service). Returns ErrNotFound if no config has been stored.
func (s *Store) GetServiceConfig(ctx context.Context, namespace, service string) (json.RawMessage, int, error) {
	if namespace == "" || service == "" {
		return nil, 0, fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidInput)
	}
	const q = `SELECT content, version FROM configs WHERE namespace = $1 AND service = $2`
	var (
		raw     []byte
		version int
	)
	if err := s.pool.QueryRow(ctx, q, namespace, service).Scan(&raw, &version); err != nil {
		return nil, 0, wrapDBError("get service config", err)
	}
	return json.RawMessage(raw), version, nil
}

// DeleteServiceConfig removes all configuration for (namespace, service).
// Returns ErrNotFound if no config exists.
func (s *Store) DeleteServiceConfig(ctx context.Context, namespace, service string) error {
	if namespace == "" || service == "" {
		return fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidInput)
	}
	const q = `DELETE FROM configs WHERE namespace = $1 AND service = $2 RETURNING namespace`
	var ns string
	return wrapDBError("delete service config", s.pool.QueryRow(ctx, q, namespace, service).Scan(&ns))
}
