# M44-Resolution-Center

## Module Metadata
- Module ID: M44
- Canonical Name: M44-Resolution-Center
- Runtime Cluster: financial-rails
- Category: Financials & Economy
- Architecture: microservice

## Primary Responsibility
Manage disputes and refund-resolution workflows, including dispute submission, messaging, approval, and event-driven workflow updates.

## Dependency Snapshot
### DBR Dependencies
- M35-Moderation-Service (owner_api)

### Event Dependencies
- payout.failed
- submission.approved

### Event Provides
- dispute.created
- dispute.resolved (analytics_only)
- transaction.refunded (listed in dependencies.yaml but removed here per canonical registry/04-services: M39 Finance is producer; M44 no longer emits)

### HTTP Provides
- yes (`POST /api/v1/disputes`, `GET /api/v1/disputes/{dispute_id}`, `POST /api/v1/disputes/{dispute_id}/messages`, `POST /api/v1/admin/disputes/{dispute_id}/approve`)

## Implementation Notes
- Internal sync dependency access uses gRPC (`M35` moderation owner API).
- Async canonical events: consumes `submission.approved` / `payout.failed`, emits `dispute.created` (domain) and `dispute.resolved` (analytics_only); `transaction.refunded` emission removed to align with 04-services canonical producer (M39 Finance).
- Mutating APIs enforce `Idempotency-Key` (7-day TTL); event consume/publish paths use 7-day dedup.
- In-memory repositories are used for mesh implementation scaffolding; transactional outbox semantics are modeled via in-memory outbox repository + worker flush.
