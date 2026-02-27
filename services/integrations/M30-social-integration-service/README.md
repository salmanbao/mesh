# M30-Social-Integration-Service

M30 Social Integration Service implementation in mesh.

## Canonical Dependencies
- DBR: `M10-Social-Integration-Verification-Service` via `owner_api`
- Consumed canonical events:
  - `social.account.connected`
  - `social.post.validated`
  - `social.compliance.violation`
  - `social.status_changed`
  - `social.followers_synced`
- Emitted canonical events: none

## API Surface
- `POST /v1/social/accounts/connect` (requires `Idempotency-Key`)
- `GET /v1/social/accounts`
- `POST /v1/social/posts/validate` (requires `Idempotency-Key`)
- `GET /v1/social/health`

## Runtime Notes
- Idempotency keys are stored for 7 days.
- Consumed event IDs are deduplicated for 7 days.
- Domain-class event processing routes failures to `social-integration-service.dlq`.
- Cross-service data reads are owner API only (M10).
