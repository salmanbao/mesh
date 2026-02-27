# M54-Analytics-Service

## Purpose
`M54-Analytics-Service` is the data-ai analytics boundary for creator/admin reporting, export generation, and canonical event ingestion into analytics warehouse facts and aggregates.

Implementation follows canonical layering:
- `internal/domain`
- `internal/application`
- `internal/ports`
- `internal/adapters`
- `internal/contracts`

## Implemented Runtime Surfaces
### REST (public edge)
- `GET /api/v1/analytics/creator/dashboard`
- `GET /api/v1/analytics/admin/financial-report`
- `POST /api/v1/analytics/export`
- `GET /api/v1/analytics/export/{id}`

### gRPC (internal sync)
- Internal server exposes health service in `internal/adapters/grpc/server.go`.
- Owner-API dependency clients are wired under `internal/adapters/grpc/clients.go`:
  - `M08-Voting-Engine`
  - `M10-Social-Integration-Verification-Service`
  - `M11-Distribution-Tracking-Service`
  - `M26-Submission-Service`
  - `M39-Finance-Service`

### Async (canonical events only)
- Worker consumes canonical event envelopes via `internal/adapters/events/worker.go`.
- Supported canonical events:
  - `submission.created`
  - `submission.approved`
  - `payout.paid`
  - `reward.calculated`
  - `campaign.launched`
  - `user.registered`
  - `transaction.succeeded`
  - `transaction.refunded`
  - `tracking.metrics.updated`
  - `discover.item_clicked`
  - `delivery.download_completed`
  - `consent.updated`
- Event deduplication by `event_id` with TTL `7 days`.
- DLQ target: `analytics-service.dlq`.

## Contract and Reliability Semantics
- Mutating REST API `POST /api/v1/analytics/export` enforces `Idempotency-Key` with TTL `7 days`.
- Canonical event handler validates required envelope fields and partition-key invariant.
- Event deduplication TTL: `7 days`.
- No cross-service direct DB writes; cross-service reads are through owner API client ports only.

## Data Ownership
Runtime persistence adapters map to canonical M54 ownership:
- `agg_earnings_daily`
- `dim_campaigns`
- `dim_users`
- `fact_clicks`
- `fact_payouts`
- `fact_submissions`
- `fact_transactions`

## Canonical Inputs Used
- `viralForge/specs/M54-Analytics-Service.md`
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/04-services.md`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `services/services-index.yaml`
