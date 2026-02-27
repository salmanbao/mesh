# M13-Escrow-Ledger-Service

Mesh implementation of the escrow and ledger microservice.

## Canonical dependencies
- Provides: `http`, `escrow.hold_created`, `escrow.partial_release`, `escrow.hold_fully_released`, `escrow.refund_processed`
- Depends on: none

## Runtime surfaces
- Public/internal edge (REST): `/v1/escrow/*`, `/v1/wallet/balance`
- Internal sync: gRPC health server placeholder
- Async: canonical escrow events via outbox relay

## Notes
- In-memory repositories are used for mesh implementation/test wiring.
- Idempotency and event dedup TTL defaults are 7 days.
- Event classes follow canonical registry: three `analytics_only`, one `domain` (`escrow.refund_processed`).
