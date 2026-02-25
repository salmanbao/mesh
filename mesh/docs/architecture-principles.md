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
