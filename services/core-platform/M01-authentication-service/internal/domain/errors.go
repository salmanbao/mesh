package domain

import "errors"

var (
	// ErrNotFound is returned when the requested resource does not exist.
	// Keeping this sentinel in domain allows adapters to map it consistently to 404/NOT_FOUND.
	ErrNotFound = errors.New("resource not found")
	// ErrInvalidCredentials hides whether email or password failed.
	// The reason is to prevent account-enumeration side channels.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrAccountLocked signals temporary lockout after repeated failed attempts.
	// This supports brute-force mitigation and a predictable user-facing response.
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
	ErrRateLimited          = errors.New("rate limited")
	// ErrOIDCFlowRequired is returned when OIDC payloads are sent to local registration endpoint.
	ErrOIDCFlowRequired = errors.New("oidc flow required")
	// ErrCannotUnlinkLastAuth prevents removing the last active authentication path.
	// Without this guard, users can lock themselves out permanently.
	ErrCannotUnlinkLastAuth = errors.New("cannot unlink last authentication method")
)
