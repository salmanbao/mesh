# M71-Integration-Hub

Phase 0 foundation-ready implementation for the M71 Integration Hub.

## Implemented Surface

- `POST /api/v1/integrations/{type}/authorize`
- `POST /api/v1/webhooks`
- `POST /api/v1/webhooks/{id}/test`
- `POST /api/v1/workflows`
- `POST /api/v1/workflows/{id}/publish`
- `POST /api/v1/workflows/{id}/test`
- compatibility aliases: `/integrations/{type}/authorize`, `/webhooks`, `/workflows`
- `POST /chat.postMessage`
- `GET /healthz`
- `GET /readyz`

## Alignment Notes

- All writes stay inside M71-owned entities represented by in-memory repositories aligned to the ownership map: integrations, API credentials, workflows, executions, webhooks, deliveries, analytics, and logs.
- Idempotency is enforced on the spec-declared mutating APIs: integration authorization, workflow creation, workflow publish, and webhook creation.
- HTTP responses use the canonical success wrapper and canonical top-level plus nested error envelope.
- No cross-service DBR or event assumptions are introduced; M71 remains dependency-consistent as an HTTP-only provider with no canonical upstream dependencies.
