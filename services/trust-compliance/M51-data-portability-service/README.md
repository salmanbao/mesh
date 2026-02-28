# M51-Data-Portability-Service

## Module Metadata
- Module ID: M51
- Canonical Name: M51-Data-Portability-Service
- Runtime Cluster: trust-compliance
- Category: Compliance & Data Governance
- Architecture: microservice

## Primary Responsibility
Handle user data export and erasure requests with idempotent APIs and auditability.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- export.completed
- export.failed

### HTTP Provides
- yes

## Canonical HTTP Endpoints
- `POST /v1/exports`
- `GET /v1/exports/{request_id}`
- `GET /v1/exports`
- `POST /v1/exports/erase`
- `GET /healthz`
- `GET /readyz`

## Ownership and Safety Notes
- Uses only M51-owned export request/audit records inside this service boundary.
- No cross-service DB reads or writes (canonical dependencies are empty).
- Mutating endpoints require `Idempotency-Key`.
- Error responses include canonical top-level fields (`code`, `message`, `request_id`) and nested `error` payload for compatibility.

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M51-*.md.
