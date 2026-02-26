# M05-Billing-Service

## Purpose
`M05-Billing-Service` is the financial-rails billing boundary for invoice creation, retrieval, delivery, voiding, refund handling, billing exports, and payout-receipt reconciliation.

Implementation follows canonical layering:
- `internal/domain`
- `internal/application`
- `internal/ports`
- `internal/adapters`
- `internal/contracts`

## Implemented Runtime Surfaces
### REST (public edge)
- `POST /v1/invoices`
- `GET /v1/invoices/{invoice_id}`
- `GET /v1/user/invoices`
- `POST /v1/invoices/{invoice_id}/send`
- `GET /v1/invoices/{invoice_id}/download`
- `GET /v1/invoices/{invoice_id}/pdf` (alias)
- `POST /v1/invoices/{invoice_id}/void`
- `GET /v1/admin/invoices`
- `GET /v1/user/billing/export`
- `POST /v1/user/billing/delete-request`
- `POST /v1/refunds`

### gRPC (internal sync)
- Internal server exposes health service in `internal/adapters/grpc/server.go`.
- Owner-API dependency clients are wired under `internal/adapters/grpc/clients.go`:
  - `M01-Authentication-Service`
  - `M09-Content-Library-Marketplace`
  - `M15-Platform-Fee-Engine`
  - `M39-Finance-Service`
  - `M61-Subscription-Service`

### Async (canonical events only)
- Consumes canonical domain events:
  - `payout.paid`
  - `payout.failed`
- Event worker: `internal/adapters/events/worker.go`
- DLQ target: `billing-service.dlq`
- Event deduplication by `event_id` with TTL `7 days`.

## Contract and Reliability Semantics
- Mutating REST APIs enforce `Idempotency-Key` with TTL `7 days`:
  - `POST /v1/invoices`
  - `POST /v1/invoices/{invoice_id}/void`
  - `POST /v1/refunds`
- Event deduplication TTL: `7 days`.
- Event-class enforcement: only `domain` class is accepted for consumed payout events.
- No cross-service direct DB writes; cross-service reads are through owner API client ports.

## Data Ownership
Owned canonical tables (modelled by service storage adapters):
- `invoice_email_events`
- `invoice_idempotency_keys`
- `invoice_line_items`
- `invoice_payments`
- `invoice_template_versions`
- `invoice_templates`
- `invoice_void_history`
- `invoices`
- `payout_receipts`
- `tax_rates`

## Canonical Inputs Used
- `viralForge/specs/M05-Billing-Service.md`
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/04-services.md`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `services/services-index.yaml`
