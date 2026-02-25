# Mesh Contracts

Contract-first artifacts for all microservices:
- `proto`: internal gRPC contracts
- `openapi`: external REST contracts
- `events`: event envelopes and schemas
- `schemas`: shared payload schemas
- `gen/go`: generated Go stubs/clients (do not edit manually)

All contract changes must preserve backward compatibility unless versioned explicitly.

## Tooling
- Buf config: `buf.yaml`, `buf.gen.yaml`
- Generate stubs: `bash scripts/contracts-buf-generate.sh --root-path .`
- Validate lint: `bash scripts/contracts-buf-lint.sh --root-path .`
- Validate compatibility: `bash scripts/contracts-buf-breaking.sh --root-path .`
