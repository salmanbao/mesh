---
name: mesh-service-audit-fix
description: Audit a mesh microservice against viralForge specs, load required specs first, then review code (REST/gRPC/events/ownership/idempotency), find defects, patch them, and run mesh gates to verify.
---

# Mesh Service Audit & Fix

Use this skill when you need to review an existing mesh microservice implementation, compare it to canonical viralForge specs, identify defects, and apply fixes.

## Source-of-truth to load first
- `viralForge/specs/service-architecture-map.yaml`
- Service spec: `viralForge/specs/Mxx-*.md`
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/04-services.md` (event registry + partition rules)
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `mesh/services/services-index.yaml`
- `mesh/tooling/manifests/implemented-services.yaml`

## Audit workflow (concise)
1) Confirm scope & location  
   - Check architecture map = microservice.  
   - Locate service under `mesh/services/<cluster>/<Mxx-service-name>/`.

2) REST contract review  
   - Compare handlers vs spec endpoints/verbs/status codes/auth.  
   - Mutations require `Idempotency-Key` (7d TTL, hash + 409 on mismatch).  
   - Ensure `X-Request-Id` enforced on mutating routes.  
   - Update `contracts/openapi/<service>.yaml` accordingly.

3) gRPC owner_api  
   - Protos in `contracts/proto/<service>/v1/`.  
   - Map domain enums to strings when populating proto fields.  
   - Expose only read/owner_api methods per DB-02 declared modes.  
   - Register servers in runtime.

4) Events  
   - Align provided/consumed events with `04-services.md`.  
   - Add `event_class` and `partition_key_path=data.<partition>` (ops -> `envelope.source_service`).  
   - Domain events use outbox+DLQ+7d dedup; analytics_only is best-effort.  
   - Update schemas under `contracts/events/*.json`; ensure publishers match payload fields.

5) Data ownership & safety  
   - No cross-service writes; reads via declared owner_api/event_projection/replica_view only.  
   - Enforce single-writer tables per service-data-ownership map.

6) Idempotency & dedup  
   - Mutations: check header, store hash w/7-day TTL, replay identical, 409 on mismatch.  
   - Webhooks use provider event IDs for dedup; events dedup by `event_id` 7 days.

7) Validation commands (repo root)  
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path .
bash mesh/scripts/validate-mesh-structure.sh --root-path .
bash mesh/scripts/gate-mesh-events-contracts.sh --root-path .
bash mesh/scripts/contracts-buf-lint.sh --root-path .
bash mesh/scripts/contracts-buf-breaking.sh --root-path .
bash mesh/scripts/contracts-buf-generate-check.sh --root-path .
bash mesh/scripts/run-mesh-gates.sh
```
Run buf commands when proto/events changed.

## Common defect checklist
- Enum/string mismatches when mapping domain → proto (cast to string).  
- Missing `event_class` / `partition_key_path` in event schemas.  
- Missing `Idempotency-Key` enforcement in middleware for mutating routes.  
- Owner API not registered or missing read-only auth checks.  
- OpenAPI missing required headers or wrong paths/verbs.  
- Outbox records not filtered by event class or lacking dedup before publish.  
- Currency/status fields not normalized (uppercase currency, string status).  

## Output expectations
- State specs consulted.  
- Summarize fixes by category (REST/gRPC/events/ownership/idempotency).  
- List files changed/artifacts generated.  
- Report gate results (pass/fail).  
- Note remaining gaps/assumptions.
