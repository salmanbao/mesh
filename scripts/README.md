# Mesh Script Reference

This folder contains all automation for mesh scaffolding, index generation, and structural gates.

## Prerequisites
- Bash 4+
- Standard Unix tools available in PATH: `find`, `sort`, `grep`, `sed`, `cmp`, `mktemp`, `wc`
- Run commands from repository root (`d:/whop-spec-docs` style path in this repo)

## Script Map
| Script | Purpose | Typical Usage |
|---|---|---|
| `libmesh.sh` | Shared parser/helpers used by other scripts. Not executed directly. | Sourced internally |
| `generate-mesh-service-scaffold.sh` | Generates/refreshes full mesh scaffold and all microservice skeletons from canonical specs. | `bash mesh/scripts/generate-mesh-service-scaffold.sh --root-path mesh` |
| `generate-mesh-index.sh` | Generates `services-index.yaml` and `docs/dependency-load-order.md` from architecture + dependencies. | `bash mesh/scripts/generate-mesh-index.sh --root-path mesh` |
| `validate-mesh-structure.sh` | Validates that required files/folders exist for all expected microservices and checks module path convention. | `bash mesh/scripts/validate-mesh-structure.sh --root-path mesh` |
| `gate-mesh1.sh` | Gate wrapper: runs structure validation + index freshness check. | `bash mesh/scripts/gate-mesh1.sh` |
| `run-mesh-gates.sh` | Entry point to run all mesh gates (currently `gate-mesh1.sh`). | `bash mesh/scripts/run-mesh-gates.sh` |

## Arguments
`generate-mesh-service-scaffold.sh`
- `--root-path` (default: `mesh`)
- `--architecture-map-path` (default: `viralForge/specs/service-architecture-map.yaml`)
- `--dependencies-path` (default: `viralForge/specs/dependencies.yaml`)
- `--deployment-profile-path` (default: `viralForge/specs/service-deployment-profile.md`)
- `--specs-dir` (default: `viralForge/specs`)

`generate-mesh-index.sh`
- `--root-path` (default: `mesh`)
- `--architecture-map-path` (default: `viralForge/specs/service-architecture-map.yaml`)
- `--dependencies-path` (default: `viralForge/specs/dependencies.yaml`)
- `--deployment-profile-path` (default: `viralForge/specs/service-deployment-profile.md`)
- `--check` (check mode; fails if generated artifacts are missing/out-of-date)

`validate-mesh-structure.sh`
- `--root-path` (default: `mesh`)
- `--architecture-map-path` (default: `viralForge/specs/service-architecture-map.yaml`)

## Recommended Workflow
1. Regenerate scaffold:
   - `bash mesh/scripts/generate-mesh-service-scaffold.sh --root-path mesh`
2. Regenerate index/docs:
   - `bash mesh/scripts/generate-mesh-index.sh --root-path mesh`
3. Validate and gate:
   - `bash mesh/scripts/run-mesh-gates.sh`

## Generated Artifacts
- `mesh/services/services-index.yaml` (machine index of microservice inventory and dependencies)
- `mesh/docs/dependency-load-order.md` (human-readable DBR topology and startup tiers)

## Failure Semantics
- Scripts use `set -euo pipefail`; any command error stops execution.
- Gate scripts exit non-zero when structural checks or freshness checks fail.

