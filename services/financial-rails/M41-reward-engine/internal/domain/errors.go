package domain

import "errors"

var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrNotFound              = errors.New("not found")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrConflict              = errors.New("conflict")
	ErrIdempotencyRequired   = errors.New("idempotency key required")
	ErrIdempotencyConflict   = errors.New("idempotency key reused with different payload")
	ErrUnsupportedEventType  = errors.New("unsupported event type")
	ErrUnsupportedEventClass = errors.New("unsupported event class")
)
