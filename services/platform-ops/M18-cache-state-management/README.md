# M18-Cache-State-Management

M18 Cache & State Management service implementation in mesh.

## Canonical Dependencies
- DBR dependencies: none
- Canonical consumed events: none
- Canonical emitted events: none

## API Surface
- `GET /v1/cache/{key}`
- `PUT /v1/cache/{key}` (requires `Idempotency-Key`)
- `DELETE /v1/cache/{key}` (requires `Idempotency-Key`)
- `POST /v1/cache/invalidate` (requires `Idempotency-Key`)
- `GET /v1/cache/metrics`
- `GET /v1/cache/health`

## Runtime Notes
- Mutating APIs enforce idempotency-key storage with 7-day TTL.
- Event dedup store is present with 7-day TTL for canonical-event handlers.
- No cross-service DB reads/writes.
- Internal sync boundary is gRPC health service; REST is the service edge.
