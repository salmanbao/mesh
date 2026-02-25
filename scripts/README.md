# Mesh Scripts Operator Guide

This folder contains all mesh automation for scaffold generation, dependency indexing, contract checks, and gate execution.

## Execution Context
- All scripts are Bash scripts and expect `set -euo pipefail` behavior.
- Prefer running from monorepo root (`d:/whop-spec-docs`) using `bash mesh/scripts/<script>.sh ...`.
- If running from `mesh/`, pass `--root-path .` where supported.

## Prerequisites
- Bash 4+
- Unix utilities in PATH: `find`, `sort`, `grep`, `sed`, `cmp`, `mktemp`, `wc`
- `git` in PATH
- `buf` in PATH for contract scripts
- Optional for code generation:
  - `protoc-gen-go`
  - `protoc-gen-go-grpc`

## Windows Notes
- Use Git Bash for execution.
- If `buf` or Go plugins are installed via `go install`, ensure `%GOPATH%\bin` is in PATH.

## Script Catalog
| Script | Type | Mutates Files | Primary Output |
|---|---|---|---|
| `libmesh.sh` | Library | No | Shared functions for other scripts |
| `generate-mesh-service-scaffold.sh` | Generator | Yes | Rewrites mesh scaffold and service skeletons |
| `generate-mesh-index.sh` | Generator/Check | Yes in normal mode, No in `--check` | `services/services-index.yaml`, `docs/dependency-load-order.md` |
| `validate-mesh-structure.sh` | Validator | No | Structural pass/fail report |
| `gate-mesh1.sh` | Gate | No | Composite check for structure + index freshness |
| `gate-mesh-events-contracts.sh` | Gate | No | Event contract coverage pass/fail for implemented services |
| `run-mesh-gates.sh` | Gate Runner | No | Full mesh gate sequence |
| `contracts-buf-lint.sh` | Validator | No | Protobuf lint result |
| `contracts-buf-breaking.sh` | Validator | No | Protobuf breaking-change result |
| `contracts-buf-generate.sh` | Generator | Yes | Generated Go stubs in `contracts/gen/go` |
| `contracts-buf-generate-check.sh` | Validator | No (fails if diff exists) | Generated artifact freshness check |
| `ci-changed-modules.sh` | CI Helper | No | JSON array of changed module paths |

## Canonical Default Paths
- Architecture map: `viralForge/specs/service-architecture-map.yaml`
- Dependencies: `viralForge/specs/dependencies.yaml`
- Deployment profile: `viralForge/specs/service-deployment-profile.md`
- Implemented services registry: `mesh/tooling/manifests/implemented-services.yaml`

## Detailed Script Documentation

### `libmesh.sh`
- Purpose: shared helper functions (parsing specs, map accumulation, sorting, YAML formatting).
- Direct usage: none; sourced by other scripts.
- Key exported helpers:
  - `load_microservices`
  - `load_dependencies`
  - `load_categories_from_profile`
  - `load_suggested_clusters_from_profile`
  - `build_clustered_maps`
  - `map_sorted_unique`, `emit_yaml_list`, `directory_name`

### `generate-mesh-service-scaffold.sh`
- Purpose: generate or refresh the entire mesh workspace skeleton deterministically.
- Reads:
  - architecture map
  - dependencies
  - deployment profile
  - service spec descriptions
- Writes:
  - root mesh files (`README.md`, `go.work`, `.env.example`)
  - `contracts/` scaffolding (`go.mod`, `buf.yaml`, `buf.gen.yaml`, readmes)
  - `platform/` module scaffold
  - all service skeleton files under `services/*/*`
  - environment compose/k8s placeholder docs
- Arguments:
  - `--root-path` default `mesh`
  - `--architecture-map-path`
  - `--dependencies-path`
  - `--deployment-profile-path`
  - `--specs-dir`
- Example:
  - `bash mesh/scripts/generate-mesh-service-scaffold.sh --root-path mesh`
- Failure modes:
  - bad canonical input paths
  - parse errors in source spec files
- Important:
  - This is a destructive refresh of scaffolded files.
  - Use in dedicated scaffold-update PRs when possible.

### `generate-mesh-index.sh`
- Purpose: generate dependency inventory and startup/load-order doc from canonical specs.
- Reads:
  - architecture map
  - dependencies
  - deployment profile
- Writes (normal mode):
  - `mesh/services/services-index.yaml`
  - `mesh/docs/dependency-load-order.md`
- Check mode:
  - `--check` compares generated temp output vs committed files and fails on drift.
- Arguments:
  - `--root-path` default `mesh`
  - `--architecture-map-path`
  - `--dependencies-path`
  - `--deployment-profile-path`
  - `--check`
- Examples:
  - Generate: `bash mesh/scripts/generate-mesh-index.sh --root-path mesh`
  - Verify freshness: `bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check`

### `validate-mesh-structure.sh`
- Purpose: validate mesh structure and service skeleton compliance.
- Reads:
  - architecture map
  - filesystem under mesh root
- Validates:
  - required root files
  - required cluster folders
  - one directory per microservice
  - service module path format in each `go.mod`
  - bootstrap package contract (`package bootstrap` and exported `Build` or `NewRuntime`)
  - k8s names/labels are lowercase RFC1123-compatible
- Arguments:
  - `--root-path` default `mesh`
  - `--architecture-map-path`
- Example:
  - `bash mesh/scripts/validate-mesh-structure.sh --root-path mesh`
- Exit:
  - non-zero with per-violation lines if any check fails.

### `gate-mesh1.sh`
- Purpose: base gate wrapper for structure and generated index drift.
- Executes:
  1. `validate-mesh-structure.sh`
  2. `generate-mesh-index.sh --check`
- Auto-detects spec root:
  - `../viralForge/specs` first
  - fallback `mesh/viralForge/specs`
- Typical usage:
  - `bash mesh/scripts/gate-mesh1.sh`

### `gate-mesh-events-contracts.sh`
- Purpose: ensure implemented services have event schema files for all consumed/emitted canonical events.
- Reads:
  - architecture map
  - dependencies
  - implemented registry
  - `contracts/events/*.json`
- Scope:
  - only services listed in `implemented-services.yaml`
- Arguments:
  - `--root-path` default `mesh`
  - `--architecture-map-path`
  - `--dependencies-path`
  - `--implemented-services-path`
  - `--contracts-events-path`
- Example:
  - `bash mesh/scripts/gate-mesh-events-contracts.sh --root-path mesh`
- Behavior:
  - if registry list is empty, script prints skip message and exits success.

### `run-mesh-gates.sh`
- Purpose: one command to run full mesh gate suite.
- Executes:
  1. `gate-mesh1.sh`
  2. `gate-mesh-events-contracts.sh`
- Typical usage:
  - `bash mesh/scripts/run-mesh-gates.sh`

### `contracts-buf-lint.sh`
- Purpose: run protobuf lint on `contracts/`.
- Arguments:
  - `--root-path` default `mesh`
- Example:
  - `bash mesh/scripts/contracts-buf-lint.sh --root-path mesh`

### `contracts-buf-breaking.sh`
- Purpose: run protobuf compatibility check against a baseline.
- Arguments:
  - `--root-path` default `mesh`
  - `--against` default `.git#branch=main,subdir=proto`
- Example:
  - `bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh --against ".git#branch=main,subdir=proto"`
- Note:
  - baseline must be reachable in the current git context.

### `contracts-buf-generate.sh`
- Purpose: generate protobuf Go code into `contracts/gen/go`.
- Arguments:
  - `--root-path` default `mesh`
- Example:
  - `bash mesh/scripts/contracts-buf-generate.sh --root-path mesh`

### `contracts-buf-generate-check.sh`
- Purpose: enforce generated code freshness.
- Flow:
  1. runs `contracts-buf-generate.sh`
  2. fails if `git diff` finds changes under `contracts/gen/go`
- Arguments:
  - `--root-path` default `mesh`
- Example:
  - `bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh`

### `ci-changed-modules.sh`
- Purpose: output changed Go module roots as JSON array for CI matrix jobs.
- Inputs:
  - positional arg1: `BASE_SHA` (optional)
  - positional arg2: `HEAD_SHA` (optional, default `HEAD`)
- Rules:
  - changes in `go.work` or `go.work.sum` fan out to all modules
  - service changes map to `services/<cluster>/<service>`
  - platform/contracts changes map to their module roots
- Output:
  - JSON array (example: `["contracts","services/core-platform/M01-authentication-service"]`)
- Example:
  - `bash mesh/scripts/ci-changed-modules.sh <base_sha> <head_sha>`

## Recommended Developer Flows

### Full maintenance flow
1. `bash mesh/scripts/generate-mesh-service-scaffold.sh --root-path mesh`
2. `bash mesh/scripts/generate-mesh-index.sh --root-path mesh`
3. `bash mesh/scripts/contracts-buf-generate.sh --root-path mesh`
4. `bash mesh/scripts/run-mesh-gates.sh`

### Verify only (no file writes expected)
1. `bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check`
2. `bash mesh/scripts/validate-mesh-structure.sh --root-path mesh`
3. `bash mesh/scripts/contracts-buf-lint.sh --root-path mesh`
4. `bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh`
5. `bash mesh/scripts/run-mesh-gates.sh`

## Troubleshooting
- `bash: command not found`:
  - use Git Bash explicitly on Windows.
- `buf: command not found`:
  - install buf and add it to PATH.
- `contracts-buf-generate-check` fails:
  - run `contracts-buf-generate.sh`, review generated diffs, commit if expected.
- `validate-mesh-structure` fails on bootstrap:
  - ensure `internal/app/bootstrap/*.go` has `package bootstrap` and exported `Build` or `NewRuntime`.
- `gate-mesh-events-contracts` fails:
  - add missing event schema JSON files in `contracts/events/`.

## Exit Semantics
- All scripts return `0` on success.
- Any validation/gate failure returns non-zero and prints actionable errors to `stderr`.
