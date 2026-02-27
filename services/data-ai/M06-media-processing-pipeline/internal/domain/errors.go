package domain

import "errors"

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrPayloadTooLarge     = errors.New("payload too large")
	ErrNotFound            = errors.New("not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrConflict            = errors.New("conflict")
	ErrIdempotencyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payload")
	ErrIdempotencyInFlight = errors.New("idempotency key request in progress")
	ErrUnsupportedEvent    = errors.New("unsupported event")
)
