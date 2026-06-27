package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// PutServiceConfig replaces the full configuration document for (namespace, service).
// content must be a valid JSON object marshaled from the service's config YAML.
// The version is incremented on each update.
func (s *Store) PutServiceConfig(ctx context.Context, namespace, service string, content json.RawMessage) error {
	if namespace == "" || service == "" {
		return fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidInput)
	}
	if len(content) == 0 {
		return fmt.Errorf("%w: content must not be empty", ErrInvalidInput)
	}
	const q = `
		INSERT INTO configs (namespace, service, content, version)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (namespace, service) DO UPDATE
			SET content    = excluded.content,
			    version    = configs.version + 1,
			    updated_at = now()`
	_, err := s.pool.Exec(ctx, q, namespace, service, content)
	return wrapDBError("put service config", err)
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
