package domain

import "errors"

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not_found")
	ErrInvalidInput        = errors.New("invalid_input")
	ErrConflict            = errors.New("conflict")
	ErrIdempotencyRequired = errors.New("idempotency_key_required")
	ErrIdempotencyConflict = errors.New("idempotency_conflict")
)
