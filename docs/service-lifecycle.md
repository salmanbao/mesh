# Mesh Service Lifecycle

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
