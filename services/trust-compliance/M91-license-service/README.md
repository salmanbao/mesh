# M91-License-Service

M91 is the license-management microservice for license listing, activation, validation, deactivation, and export.

## HTTP surface
- `GET /health`
- `GET /licenses`
- `GET /licenses/validate`
- `GET /validate`
- `POST /licenses/activate`
- `POST /activate`
- `POST /licenses/deactivate`
- `POST /licenses/exports`

## Contract rules
- Dedicated single-writer ownership over M91 license tables only.
- `Idempotency-Key` is enforced on mutating POST operations.
- Errors return the canonical top-level and nested error envelope (`status`, `code`, `message`, `request_id`, `error`).
- No cross-service DB reads or event-contract dependencies are assumed by the implementation.
