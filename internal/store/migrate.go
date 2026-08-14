package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// crdbOnlySuffix marks a migration filename as using CockroachDB-specific
// SQL (e.g. row-level TTL storage parameters) that plain Postgres rejects
// outright. The same migration files run against both a real CockroachDB
// (production, and the crdb_integration test suite) and plain Postgres (the
// faster "integration" test suite, used only for correctness that doesn't
// depend on CockroachDB-specific behavior) — a migration needing the former
// must be skipped on the latter rather than applied and failing.
const crdbOnlySuffix = "_crdb.sql"

// migrate creates the schema_migrations tracking table if absent, then applies
// any SQL files in migrations/ that have not yet been recorded. Files are applied
// in lexicographic order, which matches the numeric prefix convention (000001_, etc.).
func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	isCRDB, err := s.isCockroachDB(ctx)
	if err != nil {
		return fmt.Errorf("detect database flavor: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, filename := range files {
		if strings.HasSuffix(filename, crdbOnlySuffix) && !isCRDB {
			continue
		}

		var already bool
		err := s.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)",
			filename,
		).Scan(&already)
		if err != nil {
			return fmt.Errorf("check migration %q: %w", filename, err)
		}
		if already {
			continue
		}

		sql, err := migrationsFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", filename, err)
		}

		if _, err := s.pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %q: %w", filename, err)
		}

		if _, err := s.pool.Exec(ctx,
			"INSERT INTO schema_migrations (filename) VALUES ($1)",
			filename,
		); err != nil {
			return fmt.Errorf("record migration %q: %w", filename, err)
		}
	}

	return nil
}

// isCockroachDB reports whether the connected database is CockroachDB (true
// production and the crdb_integration suite) rather than plain Postgres
// (the faster "integration" suite) — CockroachDB implements the Postgres
// wire protocol but identifies itself distinctly in SELECT version().
func (s *Store) isCockroachDB(ctx context.Context) (bool, error) {
	var version string
	if err := s.pool.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		return false, fmt.Errorf("query version: %w", err)
	}
	return strings.Contains(version, "CockroachDB"), nil
}
