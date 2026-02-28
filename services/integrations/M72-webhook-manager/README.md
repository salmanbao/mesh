# M72-Webhook-Manager

M72 is the dedicated webhook registration and delivery-management microservice for the integrations cluster.

## HTTP surface
- `POST /api/v1/webhooks`
- `GET /api/v1/webhooks/{id}`
- `PATCH /api/v1/webhooks/{id}`
- `DELETE /api/v1/webhooks/{id}`
- `POST /api/v1/webhooks/{id}/test`
- `GET /api/v1/webhooks/{id}/deliveries`
- `GET /api/v1/webhooks/{id}/analytics`
- `POST /api/v1/webhooks/{id}/enable`
- `POST /webhook` compatibility ingress alias

## Contract rules
- Dedicated single-writer ownership over M72 webhook tables only.
- `Idempotency-Key` is enforced on mutating POST/PATCH actions implemented by this service.
- Errors return the canonical top-level and nested error envelope (`status`, `code`, `message`, `request_id`, `error`).
- No cross-service DB reads or event-contract dependencies are assumed by the implementation.
