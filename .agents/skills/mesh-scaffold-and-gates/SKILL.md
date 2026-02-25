---
name: mesh-scaffold-and-gates
description: Scaffold mesh microservices and run mesh automation scripts for index generation, structure validation, and gates. Use when a request asks to create service skeletons, refresh mesh inventory/docs, verify structure, or execute mesh quality gates.
---

# Mesh Scaffold and Gates

Use deterministic mesh scripts instead of manual structure creation whenever possible.

## 1) Load current automation contract
- Read `mesh/scripts/README.md` before execution.
- Confirm implemented-service state in `mesh/tooling/manifests/implemented-services.yaml`.
- Confirm lifecycle intent in `mesh/docs/service-lifecycle.md`.

## 2) Gather required scaffold inputs
- Identify target service ID/name and target cluster:
  - `core-platform`
  - `integrations`
  - `trust-compliance`
  - `data-ai`
  - `financial-rails`
  - `platform-ops`
- Confirm the service is classified as `architecture: microservice` in `viralForge/specs/service-architecture-map.yaml`.

## 3) Generate scaffold
- Run from repository root:
```bash
bash mesh/scripts/generate-mesh-service-scaffold.sh --root-path mesh
```
- If arguments are required by the script, provide values derived from canonical specs.
- Generator defaults to Go `1.23`, lowercase k8s-safe service names, and service-local compose build context (`deploy/compose -> ../..`).

## 4) Regenerate index and dependency docs
- Run:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
```
- Use generated artifacts as operational source of truth:
  - `mesh/services/services-index.yaml`
  - `mesh/docs/dependency-load-order.md`

## 5) Validate structure
- Run:
```bash
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
```
- Fix structural failures before proceeding.
- Validator enforces bootstrap package contract (directory exists, `package bootstrap`, exported `Build` or `NewRuntime`) and does not require a specific filename.

## 6) Run mesh gates when requested
- Run:
```bash
bash mesh/scripts/run-mesh-gates.sh
```
- `run-mesh-gates.sh` executes:
  - `gate-mesh1.sh` (structure + index drift)
  - `gate-mesh-events-contracts.sh` (strict event schema coverage for implemented services)
- Summarize failures by file and action needed.

## 7) Run contract pipeline when contracts are touched
- Run:
```bash
bash mesh/scripts/contracts-buf-lint.sh --root-path mesh
bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh
bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh
```
- If generate-check fails, run `contracts-buf-generate.sh`, review, and commit generated artifacts.

## 8) Preserve boundaries while fixing failures
- Do not move monolith-only services into `mesh`.
- Do not add cross-service direct DB writes.
- Do not bypass canonical contracts with ad-hoc payloads.
- If canonical specs were edited as part of scaffolding changes, run `viralForge/gates/run-all-gates.ps1`.

## Output expectations
- Report exactly which scripts were run.
- Report pass/fail status for each script.
- List follow-up edits needed when gates fail.
