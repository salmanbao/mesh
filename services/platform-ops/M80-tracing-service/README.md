# M80-Tracing-Service

## Module Metadata
- Module ID: M80
- Canonical Name: M80-Tracing-Service
- Runtime Cluster: platform-ops
- Category: Operational & Infrastructure
- Architecture: microservice

## Primary Responsibility
Accept tracing spans over REST ingestion endpoints, store/query trace timelines, manage sampling policies, and create trace export jobs.

## Canonical Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none (canonical async inputs not declared)

### Event Provides
- none (module-internal tracing events from spec are not canonical mesh contracts)

### HTTP Provides
- yes

## Mesh Implementation Notes
- Public edge is REST.
- Internal sync runtime includes gRPC health server only (business proto surface not authored yet).
- Canonical event handler validates envelope + partition key + 7-day dedup, then rejects unsupported event types (M80 has no canonical events declared).
- Idempotency enforced for mutating endpoints requiring `Idempotency-Key` (`POST /sampling-policies`, `POST /exports`) with 7-day TTL.
- Persistence/external dependencies are modeled with in-memory repositories and stubbed infra behavior.
