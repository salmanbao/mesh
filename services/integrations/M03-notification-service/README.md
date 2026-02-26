# M03-Notification-Service

Notification fan-out and inbox API service in the `integrations` cluster.

## Contract Snapshot

- Architecture: `microservice`
- Public interface: REST (`/v1/notifications/*`)
- Internal sync interface: gRPC (service-local, to be implemented under `internal/adapters/grpc`)
- Canonical async dependencies (consumed):
  - `auth.2fa.required`
  - `campaign.budget_updated`
  - `campaign.created`
  - `campaign.launched`
  - `dispute.created`
  - `payout.failed`
  - `payout.paid`
  - `submission.approved`
  - `submission.rejected`
  - `transaction.failed`
  - `user.registered`
- Canonical async provided events: none declared in `viralForge/specs/dependencies.yaml`.

## Ownership and Data Access

- Owned canonical tables: none (`service-data-ownership-map.yaml` and `DB-01-Data-Contracts.md`).
- Cross-service direct DB writes: forbidden.
- Cross-service reads: no declared DBR dependencies for M03; integration is event-driven.

## Reliability and Semantics

- Event deduplication requirement: 7-day TTL.
- API idempotency requirement: 7-day TTL for mutation endpoints using idempotency keys.
- Domain-event handling requirement: enforce canonical envelope and partition-key invariant.
