# M05-Billing-Service

## 1) Purpose And Scope
`M05-Billing-Service` is the financial-rails billing boundary for invoices, tax handling, payout receipts, and billing-history workflows.

Current repository state is `scaffold-only` for runtime behavior:
- service skeleton exists under `services/financial-rails/M05-billing-service/`
- business use-cases, HTTP handlers, and adapter implementations are not yet present

Canonical target behavior is defined in:
- `viralForge/specs/M05-Billing-Service.md`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`

## 2) Inbound Interfaces (Current Runtime)
### REST
- `cmd/api/main.go` prints `M05-Billing-Service API placeholder` and exits.
- `internal/adapters/http/` currently contains only `.gitkeep`.
- No runtime route registrations exist yet.

OpenAPI status:
- `contracts/openapi/m05-billing-service.yaml` documents the current runtime endpoint inventory (empty) and explicitly flags canonical-spec mismatch.

### gRPC
- No gRPC server or handler wiring is implemented yet.

### Events
- Canonical consumed events are `payout.failed` and `payout.paid` (from M14), but no consumer runtime is implemented yet.

## 3) Outbound Dependencies
Canonical DBR dependencies (owner API mode):
- `M01-Authentication-Service`
- `M09-Content-Library-Marketplace`
- `M15-Platform-Fee-Engine`
- `M39-Finance-Service`
- `M61-Subscription-Service`

Current runtime implementation:
- No outbound HTTP/gRPC/event adapter calls are wired yet.

## 4) Data Ownership And Storage Constraints
Canonical ownership model:
- architecture: `microservice`
- topology: `dedicated_service_database`
- logical ownership: `single-writer`

Owned canonical tables:
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

Constraints:
- owner-only writes for all listed M05 tables
- cross-service reads/writes must follow DBR contracts and access modes from DB-02

## 5) Control Flow (Current Code)
API process:
1. `cmd/api/main.go` executes.
2. Placeholder message is written to stdout.
3. Process exits.

Worker process:
1. `cmd/worker/main.go` executes.
2. Placeholder message is written to stdout.
3. Process exits.

Bootstrap package:
- `internal/app/bootstrap/bootstrap.go` exposes `Build() error` and returns `nil`.
- No dependency init, listeners, or workers are started yet.

## 6) Reliability Semantics
Current code:
- no idempotency store
- no retry/backoff implementation
- no outbox publisher
- no DLQ handling

Canonical requirements for future implementation are defined in:
- `viralForge/specs/M05-Billing-Service.md` (failure semantics)
- `viralForge/04-services.md` (event classes, outbox/DLQ defaults)

## 7) Security Model And Auth Assumptions
Current code:
- no auth middleware
- no JWT validation
- no authorization checks

Canonical target:
- bearer JWT auth on protected billing endpoints
- request correlation (`X-Request-Id`) for mutating operations
- audit-safe logging and PII redaction

## 8) Testing Strategy And Edge Cases
Current state:
- `tests/unit`, `tests/integration`, and `tests/contract` contain scaffold placeholders only.
- No executable service-specific assertions yet.

When implementation begins, minimum coverage should include:
- idempotent invoice creation paths
- auth-protected endpoint behavior
- payout event dedup behavior
- owner-api dependency failure and retry behavior

## 9) Decision Rationale
### Decision
Keep M05 runtime in scaffold-only mode until canonical contract and service logic rollout are explicitly scheduled.

### Context
- canonical spec is detailed and high-risk (financial workflows, invoices, refunds, payout receipts)
- partial implementation without full contract safety would create drift and false confidence
- ownership and event semantics require coordinated implementation across adapters and tests

### Alternatives Considered
- Option A: implement thin placeholder endpoints now. Not chosen because it would expose unstable contract surfaces.
- Option B: implement internal logic first without public routes. Not chosen because end-to-end behavior and failure semantics cannot be validated in isolation.

### Tradeoffs
- Benefit: avoids premature API commitments and contract drift.
- Cost: no executable billing runtime yet for dependent integration testing.

### Consequences
- Immediate: M05 remains deployable only as a scaffold placeholder.
- Long-term: implementation must land as a coordinated slice (routes, application layer, adapters, tests, contract docs) to avoid rework.

### Evidence
- `services/financial-rails/M05-billing-service/cmd/api/main.go`
- `services/financial-rails/M05-billing-service/cmd/worker/main.go`
- `services/financial-rails/M05-billing-service/internal/app/bootstrap/bootstrap.go`
- `contracts/openapi/m05-billing-service.yaml`

## 10) Canonical References
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
