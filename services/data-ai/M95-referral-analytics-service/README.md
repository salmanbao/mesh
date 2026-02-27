# M95-Referral-Analytics-Service

## Module Metadata
- Module ID: M95
- Canonical Name: M95-Referral-Analytics-Service
- Runtime Cluster: data-ai
- Architecture: microservice

## Primary Responsibility
Serve referral funnel/leaderboard/cohort/geo/forecast analytics and export jobs using M95-owned referral aggregate tables.

## Dependency Snapshot
### DBR Dependencies (owner_api)
- M89-Affiliate-Service

### Event Dependencies
- none (canonical `dependencies.yaml` declares none)

### Event Provides
- none (canonical `dependencies.yaml` declares none)

### HTTP Provides
- yes

## Implementation Notes
- Internal synchronous calls: gRPC client ports (stubbed owner-api adapter for M89).
- Public edge: REST endpoints under `/api/v1/referral-analytics/*`.
- Async canonical event handler validates envelope semantics and returns unsupported because M95 has no canonical event deps/provides.
