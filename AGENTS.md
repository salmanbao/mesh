---
name: "Mesh Microservices Implementation Guide"
description: "Repository-specific coding instructions for implementing ViralForge microservices in mesh with boundaries aligned to Solomon."
category: "Backend Service"
lastUpdated: "2026-02-17"
---

# Mesh Microservices Implementation Guide

## Mission
Implement only the services classified as `architecture: microservice` in `viralForge/specs/service-architecture-map.yaml` inside `mesh`.

This guide is intentionally aligned with Solomon module conventions so both runtimes use the same engineering model:
- `domain -> application -> ports -> adapters -> contracts`
- clear ownership boundaries
- contract-first integration

## Scope and Boundaries

### In scope (`mesh`)
- Services under `mesh/services/<cluster>/<Mxx-service-name>/`
- Technical shared code under `mesh/platform`
- Cross-service contracts under `mesh/contracts`
- Microservice automation and gates under `mesh/scripts`

### Out of scope (`mesh`)
- Monolith services implemented in `solomon`
- Direct edits to Solomon runtime code unless explicitly requested
- Cross-service direct DB write/read shortcuts that bypass contracts

### Solomon Alignment (Required)
Use the same module layering discipline as Solomon contexts:
- Solomon module: `contexts/<context>/<service>/{domain,application,ports,adapters,contracts}`
- Mesh module: `services/<cluster>/<service>/internal/{domain,application,ports,adapters,contracts}`

The layers are equivalent by intent; only the runtime packaging differs.

## Source of Truth
Always read and follow these before implementation:
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- Service spec: `viralForge/specs/Mxx-*.md`
- Mesh index: `mesh/services/services-index.yaml`

## Mesh Structure Rules

### Service location
- Each microservice must exist exactly once:
  - `mesh/services/<cluster>/<Mxx-kebab-service-name>/`
- Clusters:
  - `core-platform`
  - `integrations`
  - `trust-compliance`
  - `data-ai`
  - `financial-rails`
  - `platform-ops`

### Required service skeleton
- `README.md`
- `go.mod`
- `cmd/api/main.go`
- `cmd/worker/main.go`
- `internal/app/bootstrap/bootstrap.go`
- `internal/domain/`
- `internal/application/`
- `internal/ports/`
- `internal/adapters/http/`
- `internal/adapters/grpc/`
- `internal/adapters/events/`
- `internal/adapters/postgres/` (or service-appropriate store adapter)
- `internal/contracts/`
- `configs/default.yaml`
- `deploy/k8s/*.yaml`
- `deploy/compose/service.compose.yaml`
- `tests/unit/`, `tests/integration/`, `tests/contract/`
- `.golangci.yml`, `Makefile`

## Communication and Contract Rules
- Internal synchronous calls: gRPC.
- External/public edges: REST.
- Async integration: canonical events from `dependencies.yaml`.
- Do not invent event names when a canonical event already exists.
- Do not introduce DB coupling across microservices.

## Data Ownership Rules
- Single-writer ownership per canonical table/service.
- No direct cross-service writes.
- Cross-boundary reads must follow declared mode:
  - `owner_api`
  - `event_projection`
  - declared `replica_view`
- Respect architecture-aware access constraints from `DB-02-Shared-Data-Surface.md`.

## Implementation Workflow
1. Identify service and confirm it is `architecture: microservice`.
2. Read its `Mxx` spec and dependency edges from `dependencies.yaml`.
3. Implement in service-local layers:
   - `domain`: entities/value objects/business rules
   - `application`: commands/queries/use cases
   - `ports`: interfaces owned by the service
   - `adapters`: HTTP/gRPC/events/persistence implementations
   - `contracts`: DTO/event payload contracts
4. Keep service README dependency snapshot consistent with canonical spec.
5. Regenerate indices and run gates.

## Commands
Run from repo root:

```bash
bash mesh/scripts/generate-mesh-service-scaffold.sh --root-path mesh
bash mesh/scripts/generate-mesh-index.sh --root-path mesh
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/run-mesh-gates.sh
```

## Definition of Done
- Service remains within `mesh` microservice boundary.
- Layering is clean (`domain` does not depend on adapters).
- Contracts and dependency signals align with canonical specs.
- No forbidden data access patterns introduced.
- `run-mesh-gates.sh` passes.

## Do Not Do
- Do not move monolith services from Solomon into mesh.
- Do not add cross-service direct DB writes.
- Do not add shared business logic to `mesh/platform`.
- Do not bypass canonical specs with ad-hoc contracts.
