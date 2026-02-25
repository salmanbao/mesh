---
name: mesh-testing-and-quality
description: Define and execute testing strategy for mesh microservices, including unit, integration, contract, and regression checks tied to canonical specs and mesh gates. Use when requests involve adding tests, validating behavior changes, preventing regressions, or preparing quality sign-off.
---

# Mesh Testing and Quality

Use this skill for verification and test hardening after service implementation changes.

## 1) Load testing source of truth
- Read:
  - `mesh/services/services-index.yaml`
  - `mesh/docs/dependency-load-order.md`
  - `mesh/docs/local-dev.md`
  - `mesh/tooling/manifests/implemented-services.yaml`
  - `mesh/scripts/README.md`
  - target service `README.md`
  - target service `tests/unit`, `tests/integration`, `tests/contract`
- Align expected behavior with canonical specs in `viralForge/specs`.

## 2) Choose the minimum sufficient test scope
- Add or update:
  - unit tests for domain/application rules
  - integration tests for adapters and persistence boundaries
  - contract tests for REST/gRPC/events crossing service boundaries
- Prioritize changed code paths and dependency edges from `services-index.yaml`.

## 3) Run deterministic validation sequence
- Service-local checks:
```bash
cd mesh/services/<cluster>/<service>
go test ./...
```
- CI matrix helper when needed:
```bash
bash mesh/scripts/ci-changed-modules.sh <base_sha> <head_sha>
```
- Workspace checks:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/run-mesh-gates.sh
```
- Contract checks when interfaces/events changed:
```bash
bash mesh/scripts/contracts-buf-lint.sh --root-path mesh
bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh
bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh
```
- If behavior depends on shared infra locally, start baseline infra:
```bash
docker compose -f mesh/environments/compose/docker-compose.base.yaml up -d
```

## 4) Enforce regression and flake resistance
- Keep tests deterministic; avoid sleeps and time-sensitive assertions when possible.
- Use stable fixtures for event payloads and API contracts.
- Cover negative paths and idempotency behavior for mutating APIs and event consumers.

## 5) Report quality state with actionability
- List tests added/updated by file.
- Report executed commands and pass/fail status.
- Identify remaining risk areas and missing test coverage.
