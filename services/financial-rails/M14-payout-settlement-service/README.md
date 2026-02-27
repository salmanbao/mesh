# M14-Payout-Settlement-Service

## Purpose
`M14-Payout-Settlement-Service` is the financial-rails payout boundary for payout scheduling and settlement workflows based on reward eligibility signals.

Implementation follows canonical layering:
- `internal/domain`
- `internal/application`
- `internal/ports`
- `internal/adapters`
- `internal/contracts`

## Implemented Runtime Surfaces
### REST (public edge)
- `POST /v1/payouts/request`
- `GET /v1/payouts/{id}`
- `GET /v1/payouts/history`

### gRPC (internal sync)
- Internal server exposes health service in `internal/adapters/grpc/server.go`.
- Owner-API dependency clients are wired under `internal/adapters/grpc/clients.go`:
  - `M01-Authentication-Service`
  - `M02-Profile-Service`
  - `M05-Billing-Service`
  - `M13-Escrow-Ledger-Service`
  - `M36-Risk-Service`
  - `M39-Finance-Service`
  - `M41-Reward-Engine`

### Async (canonical events only)
- Consumes canonical domain event:
  - `reward.payout_eligible`
- Emits canonical events:
  - `payout.processing` (`analytics_only`)
  - `payout.paid` (`domain`)
  - `payout.failed` (`domain`)
- Event worker: `internal/adapters/events/worker.go`
- DLQ target: `payout-engine.dlq`
- Event deduplication by `event_id` with TTL `7 days`.

## Contract and Reliability Semantics
- Mutating REST API `POST /v1/payouts/request` enforces `Idempotency-Key` with TTL `7 days`.
- Event deduplication TTL: `7 days`.
- Domain event consume path validates canonical envelope and partition-key invariant.
- `payout.processing` is emitted via analytics publisher (no outbox/DLQ).
- Domain-class emitted events (`payout.paid`, `payout.failed`) use outbox queue + relay.
- No cross-service direct DB writes; cross-service reads are through owner API client ports.

## Data Ownership
Canonical ownership docs declare no persistent owned tables for M14. Runtime storage adapters in this implementation are in-memory abstractions for payout state, idempotency, dedup, and outbox semantics.

## Canonical Inputs Used
- `viralForge/specs/M14-Payout-Settlement-Service.md`
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/04-services.md`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `services/services-index.yaml`
