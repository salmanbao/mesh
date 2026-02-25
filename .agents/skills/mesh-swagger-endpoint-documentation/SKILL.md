---
name: mesh-swagger-endpoint-documentation
description: Create and maintain complete Swagger/OpenAPI documentation for mesh REST endpoints, including request bodies, response bodies, parameters, auth requirements, and error envelopes. Use when requests involve adding or updating API docs for service handlers/routes.
---

# Mesh Swagger Endpoint Documentation

Use this skill to document REST endpoints in OpenAPI with full request/response coverage and behavior-aligned error contracts.

## 1) Load source of truth before editing docs
- Read:
  - target service router and handlers (for method/path/auth and behavior)
  - target service request/response DTOs in `internal/application`
  - target service HTTP response helpers and error mapping middleware
  - target service README endpoint list
  - existing `mesh/contracts/openapi/*.yaml` file for that service
  - canonical service spec `viralForge/specs/Mxx-*.md`

## 2) Build endpoint inventory from runtime code
- Derive endpoint list from router registration, not from memory.
- Capture per operation:
  - method + path
  - auth requirement (public vs bearer token)
  - path/query/header parameters
  - expected status codes
  - handler-level request and response shape

## 3) Define complete OpenAPI components
- Document schemas for:
  - request bodies
  - success response envelopes
  - error response envelope
- Reuse shared components where possible (`Error`, `SuccessMessage`, `SuccessData<T>` style pattern).
- Reflect real envelope shape from code (for example `status`, `data`, `message`, `code`).

## 4) Document every operation completely
- For each path + method include:
  - `summary`, `description`, `operationId`, `tags`
  - parameters with type/format and `required`
  - `requestBody` with `application/json`, required fields, and example payloads
  - responses with explicit body schemas and examples for:
    - success (`2xx`)
    - validation/auth/domain failures (`4xx`)
    - unexpected failures (`5xx`)
- Include bearer auth scheme and apply `security` on protected routes.

## 5) Keep docs aligned with behavior
- Do not invent fields or statuses not present in handlers/service.
- If docs reveal gaps, update code/tests or flag the mismatch clearly.
- Keep OpenAPI file colocated in `mesh/contracts/openapi/` using service naming convention.

## 6) Run verification after updates
- Service-level validation:
```bash
cd mesh/services/<cluster>/<service>
go test ./...
```
- Mesh-level checks:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/run-mesh-gates.sh
```

## Output expectations
- List documented endpoints and files changed.
- Call out any handler/spec mismatches discovered.
- Report validation commands executed and pass/fail status.
