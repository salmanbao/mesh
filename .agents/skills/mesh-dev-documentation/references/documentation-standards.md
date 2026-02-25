# Documentation Standards (Mesh)

## Audience
Write for backend developers onboarding or changing a mesh service.

## Required Sections For Service Docs
1. Purpose and scope.
2. Inbound interfaces (REST/gRPC/events).
3. Outbound dependencies (DBR, upstream APIs, produced events).
4. Data ownership and storage constraints.
5. Control flow (request and async paths).
6. Reliability semantics (idempotency, retries, DLQ/outbox).
7. Security model and authorization assumptions.
8. Testing strategy and known edge cases.
9. Decision rationale and alternatives considered.

## Writing Rules
- Prefer precise statements over generic summaries.
- Use concrete paths and component names.
- Document invariants and failure behavior explicitly.
- Keep headings stable so future updates are diff-friendly.

## Canonical References
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`