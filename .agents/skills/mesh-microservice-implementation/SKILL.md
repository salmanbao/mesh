---
name: mesh-microservice-implementation
description: Implement and modify Go microservices inside mesh using canonical ViralForge specs, strict layer boundaries (domain/application/ports/adapters/contracts), and ownership-safe integration rules. Use when a request involves service logic, handlers, repositories, API/event contracts, failure semantics, or dependency-aligned behavior in mesh.
---

# Mesh Microservice Implementation

Follow this workflow for service-level coding tasks.

## 1) Confirm scope and service classification
- Read `viralForge/specs/service-architecture-map.yaml`.
- Proceed only if target service is `architecture: microservice`.
- Keep implementation under `mesh/services/<cluster>/<Mxx-service-name>/`.

## 2) Read canonical inputs before coding
- Read:
  - `viralForge/specs/Mxx-*.md` for the target service
  - `viralForge/specs/00-Canonical-Structure.md`
  - `viralForge/04-services.md`
  - `viralForge/specs/dependencies.yaml`
  - `viralForge/specs/service-data-ownership-map.yaml`
  - `viralForge/specs/DB-01-Data-Contracts.md`
  - `viralForge/specs/DB-02-Shared-Data-Surface.md`
  - `mesh/services/services-index.yaml`

## 3) Enforce architecture and runtime boundaries
- Keep service layers under `internal/`:
  - `domain`
  - `application`
  - `ports`
  - `adapters`
  - `contracts`
- Do not let `domain` depend on adapters.
- Use repository or ORM abstractions in runtime code (default: GORM from canonical structure defaults).
- Use internal sync calls via gRPC, public edges via REST, async edges via canonical events.
- Do not introduce cross-service direct DB writes or undeclared DB read paths.

## 4) Implement contract and failure semantics
- Reuse canonical event names from `dependencies.yaml` and registry in `viralForge/04-services.md`.
- Preserve event class behavior:
  - `domain`: transactional outbox + DLQ + 7-day dedup.
  - `analytics_only`: no outbox or DLQ; best-effort-deduped path to analytics.
  - `ops`: publish to `platform.audit-events` with `partition_key_path=envelope.source_service`.
- Enforce idempotency:
  - Mutating APIs use `Idempotency-Key` with 7-day TTL storage.
  - Event handlers deduplicate by `event_id` for 7 days.

## 5) Implement in ownership-safe manner
- Respect single-writer ownership for canonical tables.
- For cross-service reads, only use declared modes:
  - `owner_api`
  - `event_projection`
  - `replica_view`
- Reject access patterns not declared in `service-data-ownership-map.yaml` and `DB-02-Shared-Data-Surface.md`.

## 6) Keep service shape and docs consistent
- Ensure required service structure and entrypoints exist:
  - `cmd/api/main.go`
  - `cmd/worker/main.go`
  - `internal/app/bootstrap/` package with exported `Build` or `NewRuntime` (filename may be `bootstrap.go`, `runtime.go`, or both)
  - `configs/default.yaml`
  - `deploy/k8s/*.yaml`
  - `deploy/compose/service.compose.yaml`
- Keep the service README dependency snapshot aligned to specs.

## 7) Validate after edits
- Run from repository root:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/run-mesh-gates.sh
```
- If protobuf/event contracts changed, also run:
```bash
bash mesh/scripts/contracts-buf-lint.sh --root-path mesh
bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh
bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh
```
- If canonical spec files were changed, run:
```powershell
cd viralForge; .\gates\run-all-gates.ps1
```
- If a service transitions from scaffold-only to implemented, update:
  - `mesh/tooling/manifests/implemented-services.yaml`
  - `mesh/docs/service-lifecycle.md` readiness status

## Output expectations
- State which canonical files were used as the source of truth.
- Call out any assumptions when specs are incomplete or ambiguous.
- Report validation or gate results and unresolved gaps.
