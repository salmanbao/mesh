---
name: mesh-contract-compliance
description: Enforce canonical event contracts, data ownership constraints, and interface compatibility for mesh microservices. Use when requests touch events, protobuf/openapi payloads, cross-service reads, DB ownership, idempotency, DLQ/outbox semantics, or contract-level validation.
---

# Mesh Contract Compliance

Use this skill when correctness depends on contract and ownership rules across service boundaries.

## 1) Load contract source of truth
- Read:
  - `viralForge/04-services.md`
  - `viralForge/specs/dependencies.yaml`
  - `viralForge/specs/service-data-ownership-map.yaml`
  - `viralForge/specs/DB-01-Data-Contracts.md`
  - `viralForge/specs/DB-02-Shared-Data-Surface.md`
  - `mesh/services/services-index.yaml`
  - `mesh/tooling/manifests/implemented-services.yaml`
  - `mesh/contracts/buf.yaml`
  - `mesh/contracts/buf.gen.yaml`

## 2) Validate event contracts
- Use canonical event names from the registry/dependencies; do not invent aliases.
- Enforce event-class behavior:
  - `domain`: transactional outbox, DLQ, dedup.
  - `analytics_only`: best-effort-deduped, no outbox, no DLQ.
  - `ops`: publish to `platform.audit-events`.
- Preserve required envelope fields and partition-key invariant (`partition_key_path` resolves to `partition_key`).
- Ensure each canonical consumed/provided event has schema coverage in `mesh/contracts/events/<event>.json`.
- Use `mesh/tooling/manifests/implemented-services.yaml` as strict-enforcement scope until full rollout.

## 3) Validate data ownership and access mode
- Keep writes strictly in owned tables.
- Allow cross-service reads only through declared modes in ownership specs:
  - `owner_api`
  - `event_projection`
  - `replica_view`
- Reject direct cross-service writes and undeclared read channels.

## 4) Validate interface compatibility
- Keep gRPC/REST contracts backward compatible unless an explicit breaking-change plan is requested.
- Keep event payload schema versioning explicit when fields evolve.
- Update service README dependency/contract notes when interface surfaces change.

## 5) Run validation commands
- Always run:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/gate-mesh-events-contracts.sh --root-path mesh
bash mesh/scripts/contracts-buf-lint.sh --root-path mesh
bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh
bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh
```
- Run `bash mesh/scripts/run-mesh-gates.sh` for mesh checks.
- If canonical specs changed, run:
```powershell
cd viralForge; .\gates\run-all-gates.ps1
```
- If breaking compatibility is intentional, require explicit versioning and migration notes before merge.

## Output expectations
- List violations by rule category: event, ownership, interface, or gate.
- Include concrete file paths and corrective action.
- Call out unresolved ambiguities in canonical specs.
