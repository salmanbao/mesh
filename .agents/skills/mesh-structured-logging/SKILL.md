---
name: mesh-structured-logging
description: Add and standardize structured logging across a mesh microservice end-to-end (bootstrap, handlers, application flows, adapters) with consistent fields, safe redaction, and actionable operational signals.
---

# Mesh Structured Logging

Use this skill when a request asks to add or improve service-wide structured logging.

## 1) Read first
- Target service files:
  - `cmd/api/main.go`, `cmd/worker/main.go`
  - `internal/app/bootstrap/*`
  - `internal/adapters/http/*`, `internal/adapters/grpc/*`, `internal/adapters/events/*`
  - `internal/application/*`
  - `internal/ports/*` (for context and metadata propagation boundaries)
- Canonical context:
  - `mesh/services/services-index.yaml`
  - target service `README.md`
- Logging conventions:
  - `references/logging-field-conventions.md`

## 2) Define the logging contract for the service
- Use JSON structured logging (`slog`-style key/value fields).
- Keep field names stable across layers.
- Required baseline fields per log event:
  - `service`, `module`, `layer`, `operation`, `outcome`
- Include request/workflow correlation where available:
  - `request_id`, `trace_id`, `user_id`, `session_id`, `event_id`
- Include timing and failure signal:
  - `duration_ms`, `error`, `error_code`

## 3) Instrument in this order
1. Runtime/bootstrap:
   - startup, dependency init, listeners, shutdown.
2. Inbound edges:
   - HTTP middleware (method/path/status/duration/request_id).
   - gRPC handlers/interceptors (method/code/duration).
   - Event consumers (topic/event_id/partition/outcome).
3. Application use-cases:
   - command/query start and completion, branch decisions, failures.
4. Outbound adapters:
   - DB/cache/event publish boundaries, retries, timeout/failure causes.

## 4) Keep boundaries clean
- Prefer passing context-derived metadata, not adapter-specific logger types, into domain/application logic.
- Do not add logging that forces domain packages to depend on adapters.
- Use helper functions for repeated field sets and message formats.

## 5) Security and noise control
- Never log secrets, tokens, passwords, full auth headers, or raw PII payloads.
- Redact sensitive values and log identifiers only when needed.
- Keep info-level logs high-signal; use debug for verbose payload details.
- Avoid duplicate logs for the same error at multiple layers without added context.

## 6) Verification
- Run service tests:
```bash
cd mesh/services/<cluster>/<service>
go test ./...
```
- Run static checks used by the service (for example `go vet`, lint profile).
- Validate logs from one happy path and one failure path include required fields.

## Output expectations
- List files changed and instrumentation points added.
- Provide the final logging field set used.
- Call out any intentionally excluded fields for safety/privacy reasons.
