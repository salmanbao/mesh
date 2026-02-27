# M19-Storage-Lifecycle-Management

## Module Metadata
- Module ID: M19
- Canonical Name: M19-Storage-Lifecycle-Management
- Runtime Cluster: platform-ops
- Category: Operational & Infrastructure
- Architecture: microservice

## Primary Responsibility
System must automatically delete raw footage files 30 days after campaign closure.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC (runtime currently exposes health-only gRPC server).
- External/public interfaces: REST.
- Canonical async events: none declared; canonical event handler validates envelope + 7-day dedup then rejects unsupported event types.
- Idempotency enforced (7-day TTL) for mutating endpoints (`POST /v1/storage/policies`, `POST /storage/move-to-glacier`, `POST /storage/schedule-deletion`).
- In-memory repositories model lifecycle state, policies, batches, and audit logs while preserving ownership boundaries.
