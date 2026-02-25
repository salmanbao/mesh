# Logging Field Conventions

## Core fields (use everywhere)
- `service`: service ID or module name (e.g., `M01-Authentication-Service`)
- `module`: package or component (e.g., `bootstrap`, `http`, `application`)
- `layer`: one of `runtime`, `adapter`, `application`, `domain`
- `operation`: stable operation name (e.g., `login`, `oidc_callback`, `db_query`)
- `outcome`: `success` or `failure`

## Correlation and identity (include when available)
- `request_id`
- `trace_id`
- `user_id`
- `session_id`
- `event_id`

## Timing and failure
- `duration_ms`
- `error` (sanitized string)
- `error_code` (stable machine-readable code)

## Message style
- Use short event-style messages:
  - `request started`
  - `request completed`
  - `dependency call failed`
- Keep detail in fields, not prose.

## Redaction rules
- Never log:
  - passwords
  - tokens/JWTs/authorization headers
  - secrets, private keys
  - full request bodies containing PII
- Prefer:
  - IDs over raw objects
  - hash/fingerprint where traceability is needed

## Severity guidance
- `INFO`: lifecycle milestones and successful high-level operations
- `WARN`: recoverable failures, retries, degraded mode
- `ERROR`: user-visible failure or lost operation
- `DEBUG`: verbose diagnostics (disabled by default in production)
