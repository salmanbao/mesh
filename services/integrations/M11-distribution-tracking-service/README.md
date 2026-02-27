# M11-Distribution-Tracking-Service

Microservice implementation for M11 distribution tracking in the `integrations` cluster.

## Scope
- Validate submitted social post URLs.
- Register tracked posts and expose tracked post state.
- Poll metrics snapshots and emit canonical tracking events.

## HTTP Endpoints
- `POST /v1/tracking/posts/validate`
- `POST /v1/tracking/posts`
- `GET /v1/tracking/posts/{id}`
- `GET /v1/tracking/posts/{id}/metrics`
- `GET /healthz`
- `GET /readyz`

## Canonical Events
- Consumes: `distribution.published`, `distribution.failed`
- Emits: `tracking.metrics.updated`, `tracking.post.archived`

## Notes
- Mutating POST endpoints require `Idempotency-Key`.
- Error responses include both canonical top-level fields (`code`, `message`, `request_id`) and nested `error` payload for compatibility.
