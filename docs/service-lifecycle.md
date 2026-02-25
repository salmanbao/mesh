# Mesh Service Lifecycle

## Readiness Levels
- `scaffold-only`: service has generated skeleton and deployment placeholders but no runtime business implementation.
- `implemented`: service has business code, contracts, and passing tests for unit/integration/contract suites.
- `production-ready`: implemented service with gate compliance, operational validation, and rollout sign-off.

## Readiness Registry
- Source of truth: `tooling/manifests/implemented-services.yaml`.
- Any service listed in `implemented_services` is subject to stricter contract and gate checks.
- Promotion checklist for adding a service to the registry:
  1. Business code implemented in `internal/{domain,application,ports,adapters}`.
  2. Contracts updated in `contracts/{proto,openapi,events,schemas}` as applicable.
  3. Service tests pass (`tests/unit`, `tests/integration`, `tests/contract`).
  4. Mesh gates pass after index regeneration.

## Add a New Microservice
1. Mark service as `microservice` in `service-architecture-map.yaml`.
2. Update dependencies in `dependencies.yaml`.
3. Run `bash scripts/generate-mesh-service-scaffold.sh` and `bash scripts/generate-mesh-index.sh`.
4. Implement contracts and tests.

## Change Contracts
1. Update protobuf/openapi/event schemas.
2. Verify backward compatibility.
3. Update service README and changelog.

## Deprecate a Microservice
1. Mark deprecated in canonical maps.
2. Define successor service.
3. Keep migration notes until full cutover.
