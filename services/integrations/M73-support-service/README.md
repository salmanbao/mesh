# M73-Support-Service

M73 is the dedicated support-ticket microservice for support intake, agent assignment, replies, and CSAT collection.

## HTTP surface
- `POST /api/v1/support/tickets`
- `GET /api/v1/support/tickets/search`
- `GET /api/v1/support/tickets/{id}`
- `PATCH /api/v1/support/tickets/{id}`
- `DELETE /api/v1/support/tickets/{id}`
- `POST /api/v1/support/tickets/{id}/replies`
- `POST /api/v1/support/tickets/{id}/csat`
- `POST /api/v1/support/admin/tickets/{id}/assign`
- `POST /api/internal/tickets/create-from-email`

## Contract rules
- Dedicated single-writer ownership over M73 support tables only.
- `Idempotency-Key` is enforced on mutating POST and PATCH operations.
- Errors return the canonical top-level and nested error envelope (`status`, `code`, `message`, `request_id`, `error`).
- No cross-service DB reads or event-contract dependencies are assumed by the implementation.
