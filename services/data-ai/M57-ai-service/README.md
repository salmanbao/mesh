# M57-AI-Service

Phase 0 foundation-ready implementation for the M57 AI Service.

## Implemented Surface

- `POST /api/v1/ai/analyze`
- `POST /api/v1/ai/batch-analyze`
- `GET /api/v1/ai/batch-status/{job_id}`
- `GET /healthz`
- `GET /readyz`

## Alignment Notes

- All mutating endpoints require `Idempotency-Key` and replay previously completed responses for matching request hashes.
- The service writes only to M57-owned entities represented by in-memory repositories aligned to the spec: predictions, batch jobs, models, feedback logs, and audit logs.
- HTTP responses use the canonical success wrapper and the canonical top-level plus nested error envelope.
- No external DBR or event assumptions are introduced; this matches the canonical dependency graph where M57 has no declared dependencies and only provides HTTP.
