---
name: mesh-release-and-operations
description: Prepare and execute mesh service release readiness and operational rollout using dependency tiers, environment configs, and deployment artifacts. Use when requests involve release checklists, rollout sequencing, operational verification, or post-deploy stabilization.
---

# Mesh Release and Operations

Use this skill for deployment readiness and operational rollout, not service feature coding.

## 1) Build release context
- Read:
  - `mesh/services/services-index.yaml`
  - `mesh/docs/dependency-load-order.md`
  - `mesh/docs/service-lifecycle.md`
  - `mesh/docs/local-dev.md`
  - `mesh/tooling/manifests/implemented-services.yaml`
  - `mesh/scripts/README.md`
  - target service `configs/default.yaml`
  - target service `deploy/k8s/*.yaml`
  - target service `deploy/compose/service.compose.yaml`

## 2) Validate release prerequisites
- Confirm dependency readiness based on DBR levels and startup tiers.
- Confirm the service is listed in `implemented-services.yaml` before production promotion.
- Confirm structure/index freshness:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/run-mesh-gates.sh
```
- If release includes contract changes, require:
```bash
bash mesh/scripts/contracts-buf-lint.sh --root-path mesh
bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh
bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh
```
- Require service-level test evidence before rollout.

## 3) Sequence rollout safely
- Roll out by startup tiers from `dependency-load-order.md`:
  - `tier0` platform-ops
  - `tier1` core-platform
  - `tier2` domain clusters
- Do not roll dependent services before owner services are healthy.

## 4) Verify operational state
- Confirm service boot health and dependency connectivity.
- Validate logs/metrics/traces are emitting as expected for new changes.
- Track any degraded dependencies and hold rollout if boundary contracts fail.

## 5) Handle rollback and incident boundaries
- Scope rollback to affected services/batches; avoid cross-cluster broad rollback.
- Preserve data ownership and contract invariants during rollback.
- Record unresolved issues with service ID, impact, and next action.

## Output expectations
- Report readiness decision per service/batch.
- Report rollout order used and current status.
- Report blockers, rollback actions (if any), and follow-up operational tasks.
