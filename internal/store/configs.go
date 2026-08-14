package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// PutServiceConfig replaces the full configuration document for (namespace, service).
// content must be a valid JSON object marshaled from the service's config YAML.
// The version is incremented on each update. repoID identifies the git
// repository (see git_repositories) this config was synced from, if any —
// pass "" for configs written outside a registered repo sync (e.g. "signet
// bundle push").
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

// PatchServiceConfig atomically replaces (namespace, service)'s config
// document with apply(current) inside a single transaction — apply sees a
// consistent snapshot of the row it's about to overwrite, and CockroachDB's
// serializable isolation guarantees a concurrent PatchServiceConfig or
// PutServiceConfig against the same row can never silently interleave with
// this one: one of the two transactions is aborted with ErrConflict rather
// than either losing the other's update. The caller should retry the whole
// operation (including re-deriving the patch from fresh state) on
// ErrConflict, not just the commit.
//
// Returns ErrNotFound if no config document exists yet for (namespace,
// service) — this method only mutates an existing document; use
// PutServiceConfig (via git/bundle sync) to create the initial one. If apply
// returns an error, the transaction is rolled back and that error is
// returned unwrapped, so the caller (which knows what apply actually does,
// e.g. applying a JSON Patch) can classify it appropriately — a malformed
// patch is not a database error.
func (s *Store) PatchServiceConfig(ctx context.Context, namespace, service string, apply func(current json.RawMessage) (json.RawMessage, error)) (version int, err error) {
	if namespace == "" || service == "" {
		return 0, fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidInput)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("patch service config: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op if already committed

	var current []byte
	if err := tx.QueryRow(ctx,
		`SELECT content FROM configs WHERE namespace = $1 AND service = $2`,
		namespace, service,
	).Scan(&current); err != nil {
		return 0, wrapDBError("patch service config: read current", err)
	}

	patched, err := apply(json.RawMessage(current))
	if err != nil {
		return 0, err
	}
	if len(patched) == 0 {
		return 0, fmt.Errorf("%w: patched content must not be empty", ErrInvalidInput)
	}

	if err := tx.QueryRow(ctx,
		`UPDATE configs SET content = $3, version = version + 1, updated_at = now()
		 WHERE namespace = $1 AND service = $2
		 RETURNING version`,
		namespace, service, patched,
	).Scan(&version); err != nil {
		return 0, wrapDBError("patch service config: write", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, wrapConflictError("patch service config: commit", err)
	}
	return version, nil
}

// wrapConflictError maps a CockroachDB serialization failure (SQLSTATE
// 40001, raised on transaction commit when it can't be placed consistently
// relative to a concurrent transaction on the same data) to ErrConflict;
// anything else falls through to wrapDBError.
func wrapConflictError(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "40001" {
		return ErrConflict
	}
	return wrapDBError(op, err)
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
