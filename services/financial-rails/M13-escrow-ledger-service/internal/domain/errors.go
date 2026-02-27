package domain

import "errors"

var (
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrNotFound              = errors.New("not found")
	ErrInvalidInput          = errors.New("invalid input")
	ErrConflict              = errors.New("conflict")
	ErrIdempotencyRequired   = errors.New("idempotency key required")
	ErrIdempotencyConflict   = errors.New("idempotency conflict")
	ErrUnsupportedEventType  = errors.New("unsupported event type")
	ErrUnsupportedEventClass = errors.New("unsupported event class")
	ErrInvalidEnvelope       = errors.New("invalid envelope")
	ErrHoldClosed            = errors.New("hold closed")
	ErrInsufficientEscrow    = errors.New("insufficient escrow balance")
)
