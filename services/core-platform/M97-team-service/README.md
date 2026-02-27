# M97-Team-Service

## Module Metadata
- Module ID: M97
- Canonical Name: M97-Team-Service
- Runtime Cluster: core-platform
- Architecture: microservice

## Primary Responsibility
Manage team creation, membership, invitations, role policies, and membership authorization checks.

## Dependency Snapshot
### DBR Dependencies (owner_api)
- none

### Event Dependencies
- none

### Event Provides
- team.created
- team.member.added
- team.member.removed
- team.invite.sent
- team.invite.accepted
- team.role.changed

### HTTP Provides
- yes

## Implementation Notes
- Internal sync: gRPC runtime is health-only in this implementation (no business proto authored yet).
- Public edge: REST endpoints under `/v1/team*` for create/details/invite/accept/membership-check.
- Async: consumes no canonical events; emits canonical `team.*` domain events through outbox with DLQ + dedup semantics.
- Persistence/integrations are in-memory adapters for mesh wiring and tests (no production Postgres/Redis/Kafka yet).
