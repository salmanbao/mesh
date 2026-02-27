# M39-Finance-Service

## Purpose
`M39-Finance-Service` is the financial-rails payment boundary for transaction processing, refund handling, balance updates, and provider webhook reconciliation.

Implementation follows canonical layering:
- `internal/domain`
- `internal/application`
- `internal/ports`
- `internal/adapters`
- `internal/contracts`

## Implemented Runtime Surfaces
### REST (public edge)
- `POST /v1/transactions`
- `GET /v1/transactions/{id}`
- `GET /v1/transactions`
- `GET /v1/balances/{userID}`
- `POST /v1/refunds`
- `POST /v1/webhooks/provider`

### gRPC (internal sync)
- Internal server exposes health service in `internal/adapters/grpc/server.go`.
- Owner-API dependency clients are wired under `internal/adapters/grpc/clients.go`:
  - `M01-Authentication-Service`
  - `M04-Campaign-Service`
  - `M09-Content-Library-Marketplace`
  - `M13-Escrow-Ledger-Service`
  - `M15-Platform-Fee-Engine`
  - `M60-Product-Service`

### Async (canonical events only)
- Consumes canonical domain events: none (MVP).
- Emits canonical events:
  - `transaction.succeeded` (`domain`)
  - `transaction.failed` (`domain`)
  - `transaction.refunded` (`domain`)
- Event worker: `internal/adapters/events/worker.go`
- DLQ target: `finance-service.dlq`
- Event deduplication by `event_id` with TTL `7 days`.

## Contract and Reliability Semantics
- Mutating REST APIs enforce `Idempotency-Key` with TTL `7 days`.
- Webhook deduplication uses `webhook_id` with TTL `7 days`.
- Event deduplication TTL: `7 days`.
- Domain-class emitted events use outbox queue + relay and include required envelope + partition key invariant (`partition_key_path=data.transaction_id`).
- No cross-service direct DB writes; cross-service reads are through owner API client ports.

## Data Ownership
Runtime persistence adapters are in-memory abstractions for canonical owned entities:
- `transactions`
- `refunds`
- `user_balances`
- `transaction_webhooks`
- idempotency/dedup/outbox state

## Canonical Inputs Used
- `viralForge/specs/M39-Finance-Service.md`
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/04-services.md`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `services/services-index.yaml`
