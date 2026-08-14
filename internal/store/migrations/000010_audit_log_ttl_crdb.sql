-- audit_log retention: design/draft.md has always claimed "CockroachDB TTL
-- on audit_log table handles retention automatically (default: 90 days)",
-- but no prior migration ever actually created it -- the table grew
-- unbounded in the code as written. This fixes that.
--
-- Expiration is computed from the audit entry's own ts column (when it was
-- recorded), not row-modification time, so the TTL reflects when the event
-- happened rather than when CockroachDB last touched the row. audit_log
-- rows are append-only and never updated after insert (see internal/audit),
-- so in practice the two would coincide -- ttl_expiration_expression is used
-- anyway to state the actual intent directly rather than rely on that
-- coincidence holding forever.
ALTER TABLE audit_log SET (
    ttl = 'on',
    ttl_expiration_expression = 'ts + INTERVAL ''90 days''',
    ttl_job_cron = '@hourly'
);
