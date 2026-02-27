# M38-Content-License-Verification

## Module Metadata
- Module ID: M38
- Canonical Name: M38-Content-License-Verification
- Runtime Cluster: trust-compliance
- Category: Moderation & Compliance
- Architecture: microservice

## Primary Responsibility
Verify submission media for copyright/license risk and expose canonical hold, appeal, and DMCA intake APIs.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Canonical HTTP Endpoints
- `POST /api/v1/license/scan`
- `POST /api/v1/license/appeal`
- `POST /api/v1/admin/dmca-takedown`
- `GET /healthz`
- `GET /readyz`

## Ownership and Safety Notes
- Uses only M38-owned entities (copyright match, holds, appeals, DMCA records, audit logs).
- No cross-service DB reads/writes (dependencies are empty in canonical `dependencies.yaml`).
- Mutating endpoints require `Idempotency-Key`.
- Error responses include canonical top-level fields (`code`, `message`, `request_id`) and nested `error` payload for compatibility.

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M38-*.md.
