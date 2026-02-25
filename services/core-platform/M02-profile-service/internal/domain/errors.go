package domain

import "errors"

var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrNotFound              = errors.New("resource not found")
	ErrConflict              = errors.New("conflict")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrIdempotencyConflict   = errors.New("idempotency conflict")
	ErrStorageUnavailable    = errors.New("storage unavailable")
	ErrDependencyUnavailable = errors.New("dependency unavailable")
)
