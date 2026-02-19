# M01 Implementation Decisions

Date: 2026-02-17
Owner: M01 Authentication Service

## Contract Freeze

- Canonical table names are used: `users`, `sessions`, `oauth_connections`, `oauth_tokens`, `login_attempts`, `roles`, `email_verification_tokens`, `password_reset_tokens`, `totp_secrets`, `backup_codes`, `two_factor_methods`.
- Operational tables required by service semantics are included: `auth_outbox`, `auth_idempotency`.
- OIDC claim verification requirements are preserved: `iss`, `aud`, `exp`, `iat`, `nonce`, `sub`, `email_verified`.

## Boundary Rules

- No direct cross-service database reads or writes.
- M01 is the single writer for all `M01-Authentication-Service.*` canonical tables.
- Downstream services consume M01 data through owner API and canonical events.

## Implementation Scope (Current)

- Implemented vertical slices:
  - Register
  - Login (+ 2FA trigger event path)
  - 2FA setup/verify
  - Password reset request/reset
  - Email verify request/verify
  - Refresh
  - Logout (current session / all sessions)
  - Session listing and login history
- OIDC authorize/callback/link/unlink with real `.well-known` discovery, token endpoint exchange, and JWKS `id_token` verification

## Security Defaults

- Password hashing: bcrypt.
- Token signing: RS256.
- Session revocation checked against Redis cache and PostgreSQL source of truth.
- Sensitive fields are never logged.
