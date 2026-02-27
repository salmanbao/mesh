# M41-Reward-Engine

## Purpose
`M41-Reward-Engine` is the financial-rails reward boundary for view-locked earnings calculation, rollover balance management, and payout-eligibility signaling.

Implementation follows canonical layering:
- `internal/domain`
- `internal/application`
- `internal/ports`
- `internal/adapters`
- `internal/contracts`

## Implemented Runtime Surfaces
### REST (public edge)
- `POST /v1/rewards/calculate`
- `GET /v1/rewards/submissions/{submission_id}`
- `GET /v1/rewards/rollovers/{user_id}`
- `GET /v1/rewards/history`

### gRPC (internal sync)
- Internal server exposes health service in `internal/adapters/grpc/server.go`.
- Owner-API dependency clients are wired under `internal/adapters/grpc/clients.go`:
  - `M01-Authentication-Service`
  - `M04-Campaign-Service`
  - `M08-Voting-Engine`
  - `M11-Distribution-Tracking-Service`
  - `M26-Submission-Service`

### Async (canonical events only)
- Consumes canonical domain events:
  - `submission.auto_approved`
  - `submission.cancelled`
  - `submission.verified`
  - `submission.view_locked`
  - `tracking.metrics.updated`
- Emits canonical domain events:
  - `reward.calculated`
  - `reward.payout_eligible`
- Event worker: `internal/adapters/events/worker.go`
- DLQ target: `reward-engine.dlq`
- Event deduplication by `event_id` with TTL `7 days`.

## Contract and Reliability Semantics
- Mutating REST API `POST /v1/rewards/calculate` enforces `Idempotency-Key` with TTL `7 days`.
- Event deduplication TTL: `7 days`.
- Domain event consume path validates canonical envelope and partition-key invariant.
- Domain-class emitted events (`reward.calculated`, `reward.payout_eligible`) use outbox queue + relay.
- No cross-service direct DB writes; cross-service reads are through owner API client ports.

## Data Ownership
Canonical owned tables represented by service-local repositories:
- `campaign_rate_tiers`
- `creator_rollover_balances`
- `earnings_audit_log`
- `flagged_1099k_creators`
- `rollover_history`
- `submission_view_snapshots`

Cross-service reads are owner-api only to:
- `M01-Authentication-Service`
- `M04-Campaign-Service`
- `M08-Voting-Engine`
- `M11-Distribution-Tracking-Service`
- `M26-Submission-Service`

## Canonical Inputs Used
- `viralForge/specs/M41-Reward-Engine.md`
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/04-services.md`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `services/services-index.yaml`
