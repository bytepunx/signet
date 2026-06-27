-- restart_locks coordinates rolling restarts across service replicas.
-- Each (namespace, service) pair allows exactly one holder at a time.
-- The token field enables safe heartbeat and release operations: only the
-- current holder (matching token) can extend or release the lock.
CREATE TABLE IF NOT EXISTS restart_locks (
    namespace    TEXT        NOT NULL,
    service      TEXT        NOT NULL,
    token        TEXT        NOT NULL,
    acquired_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (namespace, service)
);
