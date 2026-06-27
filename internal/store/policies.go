package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Policy grants a SPIFFE identity access to secrets matching a glob pattern
// within a namespace.
type Policy struct {
	ID          string
	SPIFFEID    string
	Namespace   string
	Pattern     string
	Permissions []string
	CreatedAt   time.Time
}

// PutPolicy inserts a new access policy. The ID and CreatedAt fields are
// populated on return.
func (s *Store) PutPolicy(ctx context.Context, p *Policy) error {
	if err := validatePolicy(p); err != nil {
		return err
	}

	const q = `
		INSERT INTO access_policies (spiffe_id, namespace, secret_pattern, permissions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := s.pool.QueryRow(ctx, q,
		p.SPIFFEID, p.Namespace, p.Pattern, p.Permissions,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return wrapDBError("put policy", err)
	}
	return nil
}

// GetPoliciesForSPIFFE returns all policies that apply to the given SPIFFE ID.
// Returns an empty slice (not ErrNotFound) when no policies are configured.
func (s *Store) GetPoliciesForSPIFFE(ctx context.Context, spiffeID string) ([]Policy, error) {
	if spiffeID == "" {
		return nil, fmt.Errorf("%w: spiffeID must not be empty", ErrInvalidInput)
	}

	const q = `
		SELECT id, spiffe_id, namespace, secret_pattern, permissions, created_at
		FROM access_policies
		WHERE spiffe_id = $1
		ORDER BY created_at`

	rows, err := s.pool.Query(ctx, q, spiffeID)
	if err != nil {
		return nil, wrapDBError("get policies", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var p Policy
		if err := rows.Scan(
			&p.ID, &p.SPIFFEID, &p.Namespace, &p.Pattern, &p.Permissions, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("get policies: scan row: %w", err)
		}
		policies = append(policies, p)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBError("get policies", err)
	}
	return policies, nil
}

// DeletePolicy removes the policy with the given ID.
// Returns ErrNotFound if no such policy exists.
func (s *Store) DeletePolicy(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}

	tag, err := s.pool.Exec(ctx,
		"DELETE FROM access_policies WHERE id = $1", id,
	)
	if err != nil {
		return wrapDBError("delete policy", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- helpers ---

func validatePolicy(p *Policy) error {
	if p == nil {
		return fmt.Errorf("%w: policy must not be nil", ErrInvalidInput)
	}
	if p.SPIFFEID == "" {
		return fmt.Errorf("%w: SPIFFEID must not be empty", ErrInvalidInput)
	}
	if p.Namespace == "" {
		return fmt.Errorf("%w: Namespace must not be empty", ErrInvalidInput)
	}
	if p.Pattern == "" {
		return fmt.Errorf("%w: Pattern must not be empty", ErrInvalidInput)
	}
	if len(p.Permissions) == 0 {
		return fmt.Errorf("%w: Permissions must not be empty", ErrInvalidInput)
	}
	return nil
}

// GetPolicyByID returns a single policy by its ID. Used internally for tests.
func (s *Store) GetPolicyByID(ctx context.Context, id string) (*Policy, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id must not be empty", ErrInvalidInput)
	}

	const q = `
		SELECT id, spiffe_id, namespace, secret_pattern, permissions, created_at
		FROM access_policies
		WHERE id = $1`

	var p Policy
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.SPIFFEID, &p.Namespace, &p.Pattern, &p.Permissions, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, wrapDBError("get policy by id", err)
	}
	return &p, nil
}
