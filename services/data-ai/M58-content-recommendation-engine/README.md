# M58-Content-Recommendation-Engine

## Module Metadata
- Module ID: M58
- Canonical Name: M58-Content-Recommendation-Engine
- Runtime Cluster: data-ai
- Category: AI & Automation
- Architecture: microservice

## Primary Responsibility
Generate role-aware content and campaign recommendations, persist recommendation/feedback/override state, and emit module-internal recommendation domain events via transactional outbox semantics.

## Dependency Snapshot
### DBR Dependencies
- M23-Campaign-Discovery-Service (owner_api)

### Event Dependencies
- none declared canonically (`dependencies.yaml`)

### Event Provides
- none declared canonically
- module-internal only (non-canonical per M58 spec): `recommendation.generated`, `recommendation.feedback_recorded`, `recommendation.override_applied`

### HTTP Provides
- yes (`GET /api/v1/recommendations`, `POST /api/v1/recommendations/{recommendation_id}/feedback`, `POST /api/v1/admin/recommendation-overrides`)

## Implementation Notes
- Internal synchronous dependency access uses gRPC client adapters (M23 only).
- Public edge uses REST handlers under `internal/adapters/http`.
- Async domain events are written to an outbox and flushed by worker; events are module-internal and not canonical registry contracts.
- API idempotency and event dedup both use 7-day TTL stores.
