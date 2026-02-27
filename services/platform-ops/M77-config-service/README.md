# M77-Config-Service

Mesh implementation of the ViralForge config service with strict layering:
`domain -> application -> ports -> adapters -> contracts`.

## Canonical Snapshot
- Service: `M77-Config-Service`
- Cluster: `platform-ops`
- Architecture: `microservice`
- Provides: `http`
- Event deps/provides: none (canonical)
- DB read dependencies: none

## Implemented Surface (REST)
- `GET /api/v1/config`
- `PATCH /api/v1/config/{key}`
- `POST /api/v1/config/import`
- `GET /api/v1/config/export`
- `POST /api/v1/config/rollback`
- `GET /api/v1/config/audit`
- `POST /api/v1/config/rollout-rules`
- `GET /health`, `GET /metrics`, `GET /healthz`, `GET /readyz`

## Runtime Notes
- Internal sync runtime includes gRPC health server only (business gRPC proto/API not yet authored).
- Async canonical event consume/emit is disabled because canonical dependencies declare no events for M77.
- Idempotency (`Idempotency-Key`) and event dedup TTL defaults are 7 days.
