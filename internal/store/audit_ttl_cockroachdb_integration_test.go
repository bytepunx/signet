//go:build crdb_integration

package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrate_AppliesAuditLogTTLOnCockroachDB verifies migration
// 000010_audit_log_ttl_crdb.sql actually takes effect: design/draft.md has
// always claimed a 90-day retention TTL on audit_log, but no migration ever
// created it before this fix, so the table grew unbounded. This confirms
// the row-level TTL storage parameters are really set on the live table,
// not just that the migration's SQL is syntactically plausible.
func TestMigrate_AppliesAuditLogTTLOnCockroachDB(t *testing.T) {
	s := newCRDBTestStore(t)
	ctx := context.Background()

	var createStmt string
	require.NoError(t, s.pool.QueryRow(ctx,
		"SELECT create_statement FROM [SHOW CREATE TABLE audit_log]",
	).Scan(&createStmt))

	assert.Contains(t, createStmt, "ttl = 'on'")
	assert.Contains(t, createStmt, "ttl_job_cron = '@hourly'")
	// CockroachDB re-escapes the expression string on echo (e'ts + INTERVAL
	// \'90 days\''), so check its substantive pieces rather than the exact
	// quoting.
	assert.Contains(t, createStmt, "ttl_expiration_expression",
		"audit_log must expire rows based on their own ts, not row-modification time")
	assert.Contains(t, createStmt, "ts + INTERVAL")
	assert.Contains(t, createStmt, "90 days")
}
