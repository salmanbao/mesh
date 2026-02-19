package domain

import "errors"

var (
	ErrNotFound             = errors.New("resource not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrAccountLocked        = errors.New("account locked")
	ErrSessionRevoked       = errors.New("session revoked")
	ErrSessionExpired       = errors.New("session expired")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrInvalidInput         = errors.New("invalid input")
	ErrNotImplemented       = errors.New("not implemented")
	ErrConflict             = errors.New("conflict")
	ErrInsufficientAuth     = errors.New("insufficient authentication factors")
	ErrIdempotencyConflict  = errors.New("idempotency conflict")
	ErrRoleResolutionFailed = errors.New("role resolution failed")
	ErrTokenExpired         = errors.New("token expired")
	ErrTokenConsumed        = errors.New("token already consumed")
	ErrCannotUnlinkLastAuth = errors.New("cannot unlink last authentication method")
)
