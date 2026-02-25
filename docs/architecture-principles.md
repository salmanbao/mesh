# Mesh Architecture Principles

## Boundary Rules
- `mesh` contains only services marked `architecture: microservice`.
- `solomon` remains the monolith runtime; no duplication of service implementation.
- Cross-service writes are prohibited.

## Communication
- Internal synchronous communication: gRPC.
- Public/external interfaces: REST.
- Asynchronous integration: canonical events from `dependencies.yaml`.

## Shared Code
- Shared code is technical only and versioned in `mesh/platform`.
- Domain/business logic must remain service-local.

## Bootstrap Entrypoint Policy
- Service bootstrap contract is package-level, not filename-level.
- Valid bootstrap implementations may use `internal/app/bootstrap/bootstrap.go`, `runtime.go`, or both.
- A service is bootstrap-compliant when:
  - `internal/app/bootstrap` exists,
  - package name is `bootstrap`,
  - at least one exported entrypoint is present (`Build` or `NewRuntime`).
