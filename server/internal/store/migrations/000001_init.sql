CREATE TABLE IF NOT EXISTS secrets (
    namespace     TEXT        NOT NULL,
    service       TEXT        NOT NULL,
    secret_name   TEXT        NOT NULL,
    version       INT         NOT NULL,
    encrypted_dek BYTEA       NOT NULL,
    ciphertext    BYTEA       NOT NULL,
    expires_at    TIMESTAMPTZ,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (namespace, service, secret_name, version)
);

CREATE TABLE IF NOT EXISTS access_policies (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    spiffe_id      TEXT        NOT NULL,
    namespace      TEXT        NOT NULL,
    secret_pattern TEXT        NOT NULL,
    permissions    TEXT[]      NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_access_policies_spiffe_id ON access_policies (spiffe_id);

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ts          TIMESTAMPTZ NOT NULL DEFAULT now(),
    spiffe_id   TEXT        NOT NULL,
    action      TEXT        NOT NULL,
    namespace   TEXT        NOT NULL,
    secret_name TEXT        NOT NULL,
    outcome     TEXT        NOT NULL,
    peer_ip     TEXT,
    hmac        BYTEA       NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_log_ts ON audit_log (ts DESC);
