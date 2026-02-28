# M70-Developer-Portal

Phase 0 foundation-ready implementation for the M70 Developer Portal.

## Implemented Surface

- `POST /api/v1/developers/register`
- `POST /api/v1/developers/api-keys`
- `POST /api/v1/developers/api-keys/{id}/rotate`
- `POST /api/v1/developers/api-keys/{id}/revoke`
- `POST /api/v1/developers/webhooks`
- `POST /api/v1/developers/webhooks/{id}/test`
- compatibility aliases: `/api-keys`, `/webhooks`
- `GET /healthz`
- `GET /readyz`

## Alignment Notes

- All writes stay inside M70-owned entities represented by in-memory repositories aligned to the ownership map: developers, sessions, API keys, key rotations, webhooks, deliveries, usage, and audit rows.
- Idempotency is enforced on the spec-declared mutating APIs: developer registration, API key creation, API key rotation, and webhook creation.
- HTTP responses use the canonical success wrapper and canonical top-level plus nested error envelope.
- No cross-service DBR or event assumptions are introduced; M70 remains dependency-consistent as an HTTP-only provider with no canonical upstream dependencies.
