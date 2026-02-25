---
name: mesh-rollout-orchestration
description: Plan and execute mesh-wide rollout across all microservices using canonical dependency order, cluster tiers, and batch checkpoints. Use when a request is about implementing many services, sequencing delivery, tracking readiness, or coordinating parallel service work in mesh.
---

# Mesh Rollout Orchestration

Use this skill for multi-service execution across the mesh program, not single-service coding.

## 1) Build the canonical rollout inventory
- Read:
  - `viralForge/specs/service-architecture-map.yaml`
  - `viralForge/specs/dependencies.yaml`
  - `mesh/services/services-index.yaml`
  - `mesh/docs/dependency-load-order.md`
  - `mesh/tooling/manifests/implemented-services.yaml`
  - `mesh/docs/service-lifecycle.md`
- Treat only `architecture: microservice` services as rollout scope.
- If index/docs are stale or missing, regenerate with `mesh/scripts/generate-mesh-index.sh`.

## 2) Sequence by dependency-safe batches
- Use DBR topological levels first, then cluster startup tiers.
- Parallelize services that are in the same level and do not depend on each other.
- Do not start a service batch that depends on unfinished owner services.
- Promote services to implemented status explicitly after passing quality gates.

## 3) Execute each batch with deterministic handoff
- For scaffold refresh or missing structure, use `mesh-scaffold-and-gates`.
- For service-local logic and contracts, use `mesh-microservice-implementation`.
- For promotion to implemented status, update `mesh/tooling/manifests/implemented-services.yaml` in the same change set.
- Keep rollout reporting per batch:
  - services attempted
  - services completed
  - blockers and missing upstreams

## 4) Run checkpoints after every batch
- Run:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/run-mesh-gates.sh
```
- If contracts changed in the batch, also run:
```bash
bash mesh/scripts/contracts-buf-lint.sh --root-path mesh
bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh
bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh
```
- Use `bash mesh/scripts/ci-changed-modules.sh <base_sha> <head_sha>` for path-based module matrix planning.

## 5) Preserve program boundaries
- Keep monolith services out of mesh implementation scope.
- Keep DB ownership and event contracts canonical; escalate spec conflicts instead of patching ad hoc.
- Keep startup/load-order docs aligned with implementation reality.

## Output expectations
- Report completed batches and current dependency frontier.
- Identify blockers with exact upstream service IDs.
- Provide next executable batch, not just a backlog list.
