-- config_path lets a single repository serve both SOPS-encrypted secrets and
-- plain-YAML config files from independent directory roots.
ALTER TABLE git_repositories ADD COLUMN IF NOT EXISTS config_path TEXT NOT NULL DEFAULT '';

-- configs stores per-service configuration as a JSONB document ingested from
-- plain (non-SOPS) YAML files. Each sync fully replaces the document for
-- (namespace, service); the version increments on every write.
CREATE TABLE IF NOT EXISTS configs (
    namespace   TEXT        NOT NULL,
    service     TEXT        NOT NULL,
    content     JSONB       NOT NULL,
    version     INT         NOT NULL DEFAULT 1,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (namespace, service)
);

CREATE INDEX IF NOT EXISTS idx_configs_ns_svc ON configs (namespace, service);
