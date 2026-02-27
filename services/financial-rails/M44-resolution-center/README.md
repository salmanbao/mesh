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
- dispute.resolved
- transaction.refunded (canonical dependency entry; note spec narrative conflicts and says M39 emits this)

### HTTP Provides
- yes (`POST /api/v1/disputes`, `GET /api/v1/disputes/{dispute_id}`, `POST /api/v1/disputes/{dispute_id}/messages`, `POST /api/v1/admin/disputes/{dispute_id}/approve`)

## Implementation Notes
- Internal sync dependency access uses gRPC (`M35` moderation owner API).
- Async canonical events: consumes `submission.approved` / `payout.failed`, emits `dispute.created` (domain), `dispute.resolved` (analytics_only), and `transaction.refunded` (domain) per `dependencies.yaml`.
- Mutating APIs enforce `Idempotency-Key` (7-day TTL); event consume/publish paths use 7-day dedup.
- In-memory repositories are used for mesh implementation scaffolding; transactional outbox semantics are modeled via in-memory outbox repository + worker flush.
