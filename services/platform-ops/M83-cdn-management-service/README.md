# M83-CDN-Management-Service

M83 is the CDN control-plane microservice for configuration versioning, purge requests, metrics, and certificate visibility.

## HTTP surface
- `GET /health`
- `GET /metrics`
- `GET /configs`
- `POST /configs`
- `POST /purge`

## Contract rules
- Dedicated single-writer ownership over M83 CDN tables only.
- `Idempotency-Key` is enforced on mutating POST operations.
- Errors return the canonical top-level and nested error envelope (`status`, `code`, `message`, `request_id`, `error`).
- No cross-service DB reads or event-contract dependencies are assumed by the implementation.
