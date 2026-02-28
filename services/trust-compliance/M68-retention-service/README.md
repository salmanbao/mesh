# M68-Retention-Service

Phase 0 foundation-ready implementation for the M68 Retention Service.

## Implemented Surface

- `GET /api/v1/retention/policies`
- `POST /api/v1/retention/policies`
- `POST /api/v1/retention/preview`
- `POST /api/v1/retention/preview/{preview_id}/approve`
- `GET /api/v1/retention/legal-holds`
- `POST /api/v1/retention/legal-holds`
- `POST /api/v1/retention/restorations`
- `POST /api/v1/retention/restorations/{restoration_id}/approve`
- `GET /api/v1/retention/reports/compliance`
- `GET /healthz`
- `GET /readyz`

## Alignment Notes

- All writes remain inside M68-owned entities represented by in-memory repositories aligned to the ownership map: policies, previews, legal holds, restorations, scheduled deletions, and audit rows.
- Idempotency is enforced on the spec-declared mutating APIs: policy creation, legal hold creation, and restoration creation.
- HTTP responses use the canonical success wrapper and canonical top-level plus nested error envelope.
- No cross-service DBR or event assumptions are introduced, preserving canonical dependency consistency for M68 as an HTTP-only provider with no declared dependencies.
