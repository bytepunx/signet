// Package store is the only package that communicates with CockroachDB.
// It never handles plaintext secrets — only ciphertext and encrypted DEKs.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store holds the database connection pool and exposes typed operations for
// secrets, policies, and audit logging.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a connection pool to the database at connStr, verifies connectivity,
// and runs any pending schema migrations. The context governs the startup phase only;
// subsequent operations use their own contexts.
func New(ctx context.Context, connStr string) (*Store, error) {
	if connStr == "" {
		return nil, fmt.Errorf("%w: connection string must not be empty", ErrInvalidInput)
	}

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

// Close releases all connections in the pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Ping verifies the database is reachable.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return nil
}
