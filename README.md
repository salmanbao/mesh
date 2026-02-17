# Mesh Microservices Workspace

`mesh` hosts all services classified as `architecture: microservice` in `viralForge/specs/service-architecture-map.yaml`.

## Scope
- Service model: one Go module per microservice.
- Runtime model: Kubernetes-first manifests with Docker Compose local parity.
- Interface model: gRPC for internal service-to-service calls, REST for external/public APIs.
- Shared technical primitives: versioned libraries under `mesh/platform`.

## Source Of Truth
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/service-deployment-profile.md`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `viralForge/specs/dependencies.yaml`

## Non-Goals
- Implementing monolith services (those remain in `solomon`).
- Overriding canonical contracts or ownership boundaries defined in specs.

## Automation Scripts
- Script location: `mesh/scripts`
- Script guide and usage: `mesh/scripts/README.md`
