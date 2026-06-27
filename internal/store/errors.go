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
)
