-- age keys managed by signet for SOPS file decryption.
-- The private key is stored encrypted under signet's envelope encryption;
-- plaintext private key material never reaches the database.
CREATE TABLE IF NOT EXISTS sops_age_keys (
    public_key            TEXT        NOT NULL PRIMARY KEY,
    encrypted_private_key BYTEA       NOT NULL,
    is_active             BOOL        NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at        TIMESTAMPTZ
);

-- Git repositories whose secret trees signet will sync.
-- Both the deploy key (SSH private key) and the webhook secret are stored
-- encrypted under signet's envelope encryption.
CREATE TABLE IF NOT EXISTS git_repositories (
    id                       UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    name                     TEXT        NOT NULL UNIQUE,
    repo_url                 TEXT        NOT NULL,
    branch                   TEXT        NOT NULL DEFAULT 'main',
    secrets_path             TEXT        NOT NULL DEFAULT 'secrets/',
    encrypted_webhook_secret BYTEA       NOT NULL,
    encrypted_deploy_key     BYTEA       NOT NULL,
    last_sync_sha            TEXT,
    last_sync_at             TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);
