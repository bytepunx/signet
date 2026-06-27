package unseal

import "errors"

var (
	// ErrAlreadyUnsealed is returned when an unseal operation is attempted on a
	// server that is already unsealed.
	ErrAlreadyUnsealed = errors.New("already unsealed")

	// ErrShamirNotConfigured is returned when SubmitShare is called but the
	// manager was not configured with a Shamir threshold.
	ErrShamirNotConfigured = errors.New("Shamir unsealing is not configured")

	// ErrSharesExpired is returned when the share accumulation window expired
	// before the threshold was reached. Accumulated shares are discarded.
	ErrSharesExpired = errors.New("share accumulation expired: threshold not reached within the configured timeout")

	// ErrInvalidShare is returned when a submitted share is malformed or
	// inconsistent with the other accumulated shares.
	ErrInvalidShare = errors.New("invalid share")

	// ErrTPMNotSupported is returned when UnsealWithTPM is called but TPM
	// support was not compiled in. Rebuild with -tags tpm to enable it.
	ErrTPMNotSupported = errors.New("TPM support not compiled in: rebuild with -tags tpm")

	// ErrInvalidConfig is returned when New is called with an invalid Config.
	ErrInvalidConfig = errors.New("invalid unseal configuration")
)
