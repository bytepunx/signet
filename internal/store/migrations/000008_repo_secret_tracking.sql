-- repo_id lets FullSync (signet's explicit "signet repo sync" path, chosen
-- deliberately over webhook-driven sync so no public cluster exposure is
-- needed) determine which secrets/configs it is currently responsible for,
-- so it can detect and clean up ones removed from the repo. Before this,
-- only the webhook-driven incremental path (which gets an explicit deleted-
-- files list from GitHub's push payload) ever deleted anything; a full
-- sync only ever added/updated, so a secret or config file removed from a
-- repo synced exclusively via "signet repo sync" was never cleaned up.
--
-- Nullable: rows written before this migration, or written via "signet
-- bundle push" (which has no registered repository at all), have no
-- attributable repo and are simply never considered for deletion-on-sync.
-- ON DELETE SET NULL rather than CASCADE: removing a repository
-- registration must not delete the secrets/configs it synced.
ALTER TABLE secrets ADD COLUMN IF NOT EXISTS repo_id UUID REFERENCES git_repositories(id) ON DELETE SET NULL;
ALTER TABLE configs ADD COLUMN IF NOT EXISTS repo_id UUID REFERENCES git_repositories(id) ON DELETE SET NULL;

-- Not a partial index (no WHERE repo_id IS NOT NULL): CockroachDB rejects a
-- partial index predicate that references a column added earlier in the
-- same migration transaction ("cannot create partial index on column ...
-- which is not public" — the column hasn't finished its online schema
-- change within this transaction yet). A plain index has no such
-- restriction.
CREATE INDEX IF NOT EXISTS idx_secrets_repo_id ON secrets (repo_id);
CREATE INDEX IF NOT EXISTS idx_configs_repo_id ON configs (repo_id);
