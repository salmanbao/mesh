# M69-Legal-Service

Phase 0 foundation-ready implementation for the M69 Legal Service.

## Implemented Surface

- `POST /api/v1/legal/documents/upload`
- `POST /api/v1/legal/documents/{id}/signatures`
- `POST /api/v1/legal/holds`
- `GET /api/v1/legal/holds/check`
- `POST /api/v1/legal/holds/{id}/release`
- `POST /api/v1/legal/compliance/scan`
- `GET /api/v1/legal/compliance/reports/{report_id}`
- `POST /api/v1/legal/disputes`
- `GET /api/v1/legal/disputes/{dispute_id}`
- `POST /api/v1/legal/dmca-notices`
- `POST /api/v1/legal/regulatory-filings/generate-1099`
- `GET /api/v1/legal/regulatory-filings/{filing_id}/status`
- compatibility aliases: `/legal/documents/upload`, `/legal/holds`, `/legal/compliance/scan`
- `GET /healthz`
- `GET /readyz`

## Alignment Notes

- All writes stay inside M69-owned entities represented by in-memory repositories aligned to the ownership map: documents, signatures, legal holds, compliance reports/findings, disputes, DMCA notices, filings, and audit rows.
- Idempotency is enforced on the spec-declared mutating APIs: document upload, hold creation, dispute creation, DMCA notice creation, and 1099 filing generation.
- HTTP responses use the canonical success wrapper and canonical top-level plus nested error envelope.
- No cross-service DBR or event assumptions are introduced; M69 remains dependency-consistent as an HTTP-only provider with no canonical upstream dependencies.
