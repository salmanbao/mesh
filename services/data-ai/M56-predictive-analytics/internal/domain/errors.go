package domain

import "errors"

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrIdempotencyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payload")
)
