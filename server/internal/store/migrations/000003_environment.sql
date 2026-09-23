-- Add environment scoping to SOPS age keys.
--
-- An empty string means the key is global (usable by any signet instance
-- regardless of its SIGNET_ENVIRONMENT setting). A non-empty value means the
-- key is scoped to that specific environment (e.g. "prod", "staging", "dev").
--
-- All existing keys default to "" (global) so there is no behavioural change
-- for deployments that have not yet configured SIGNET_ENVIRONMENT.
ALTER TABLE sops_age_keys
    ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT '';
