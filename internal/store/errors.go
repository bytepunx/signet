package store

import "errors"

var (
	// ErrNotFound is returned when a requested record does not exist.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is returned when an insert violates a unique constraint.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidInput is returned when a required field is missing or malformed.
	// Callers can use errors.Is to distinguish programming errors from database errors.
	ErrInvalidInput = errors.New("invalid input")

	// ErrConflict is returned when a transaction is aborted because it
	// serialized after a concurrent transaction that touched the same row
	// (CockroachDB SQLSTATE 40001). The operation was not lost — nothing was
	// written — the caller should simply retry the whole operation, which
	// will see the concurrent change on its next attempt. See
	// Store.PatchServiceConfig.
	ErrConflict = errors.New("conflict: concurrent modification, retry")
)
