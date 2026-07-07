-- Introduces the key-encryption-key (KEK) tier of the envelope encryption
-- hierarchy: Master -> KEK -> DEK -> secret. Wrapping DEKs under a KEK
-- (instead of directly under the master key) allows rotating the KEK by
-- re-wrapping every DEK (cheap) without re-encrypting any secret blob, and
-- allows rotating the master key by re-wrapping only the KEK rows (even
-- cheaper) without touching secrets at all.
--
-- secrets.kek_id is nullable for backward compatibility: rows written before
-- this migration have their DEK wrapped directly under the master key
-- (kek_id IS NULL). Decryption code must treat NULL as "unwrap directly with
-- the master key" and only look up key_encryption_keys when kek_id is set.
CREATE TABLE IF NOT EXISTS key_encryption_keys (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    wrapped_kek    BYTEA       NOT NULL,
    is_active      BOOLEAN     NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at TIMESTAMPTZ
);

ALTER TABLE secrets ADD COLUMN IF NOT EXISTS kek_id UUID REFERENCES key_encryption_keys(id);

-- A key-check value lets the server verify a supplied master key is correct
-- immediately after unseal, before declaring the server operational. Single
-- row keyed by a fixed id.
CREATE TABLE IF NOT EXISTS key_check_value (
    id         TEXT        PRIMARY KEY DEFAULT 'singleton',
    ciphertext BYTEA       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Repository names are looked up by GetRepositoryByName and are used to bind
-- the AAD context for the repo's encrypted webhook secret and deploy key;
-- both uses assume uniqueness, so make it an enforced constraint.
CREATE UNIQUE INDEX IF NOT EXISTS idx_git_repositories_name ON git_repositories (name);
