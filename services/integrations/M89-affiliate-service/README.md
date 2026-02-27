# M89-Affiliate-Service

Mesh implementation of the Affiliate Service with clean layering under `internal/{domain,application,ports,adapters,contracts}`.

## Scope
- Public REST for affiliate link management, dashboard, earnings, exports, and admin controls
- Internal gRPC (health-only runtime stub)
- Canonical event publishing:
  - `affiliate.click.tracked`
  - `affiliate.attribution.created`

## Canonical Alignment
- Source of truth: `viralForge/specs/M89-Affiliate-Service.md`, `viralForge/specs/dependencies.yaml`, `viralForge/specs/service-data-ownership-map.yaml`
- Canonical dependencies declare `provides: [EVENT:affiliate.click.tracked, EVENT:affiliate.attribution.created, http]`
- Canonical dependencies declare no inbound event dependencies for M89, so async inbound handler validates envelope/dedup and rejects unsupported events

## Storage / Ownership
In-memory repositories model M89-owned tables only:
- `affiliates`
- `referral_links`
- `referral_clicks`
- `referral_attributions`
- `affiliate_earnings`
- `affiliate_payouts`
- `affiliate_audit_logs`

Service-local support stores:
- idempotency keys (7-day TTL)
- event dedup (7-day TTL)
- outbox

## Local Run
```bash
go test ./...
go run ./cmd/api
go run ./cmd/worker
```

## Notes
- External integrations (payments, Kafka broker, durable Postgres) are stubbed with in-memory adapters in this mesh implementation.
- Shared OpenAPI contract artifact is not included yet (`contracts/openapi/m89-affiliate-service.yaml`).
