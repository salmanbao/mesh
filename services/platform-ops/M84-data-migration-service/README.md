# M84-Data-Migration-Service

M84 is the migration-control microservice for plan validation, zero-downtime execution orchestration, registry updates, and backfill tracking.

## HTTP surface
- `GET /health`
- `GET /plans`
- `POST /plans`
- `POST /runs`

## Contract rules
- Dedicated single-writer ownership over M84 migration tables only.
- `Idempotency-Key` is enforced on mutating POST operations.
- Migration execution (`POST /runs`) requires operator role plus `X-MFA-Verified: true`.
- Errors return the canonical top-level and nested error envelope (`status`, `code`, `message`, `request_id`, `error`).
- No cross-service DB reads or event-contract dependencies are assumed by the implementation.
