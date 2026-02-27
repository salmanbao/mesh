---
name: mesh-spec-contract-remediator
description: Audit a mesh microservice against viralForge specs; fix REST/gRPC/event contracts, DBR access, regenerate artifacts, and wire integrations end-to-end.
---

# Mesh Spec Contract Remediator

Use this skill when a mesh microservice must be reconciled to the canonical viralForge specs (`../viralForge/specs`). It audits and fixes REST, gRPC owner_api, canonical events, and data ownership, then regenerates required contract artifacts and runs gates.

## Load source of truth first
- `viralForge/specs/service-architecture-map.yaml` (architecture == microservice)
- `viralForge/specs/Mxx-*.md` for the target service
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/04-services.md` (canonical event registry & partition keys)
- `viralForge/specs/dependencies.yaml` (provides/depends_on including events, DBR)
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `mesh/services/services-index.yaml`
- `mesh/tooling/manifests/implemented-services.yaml` (strict event enforcement scope)

## Audit checklist
1. Scope  
   - Service exists under `mesh/services/<cluster>/<service>/` and is marked `architecture: microservice`.
2. REST surface  
   - Compare spec endpoints to `internal/adapters/http/*` and `contracts/openapi/<service>.yaml`.  
   - Add missing paths/verbs/status codes and required auth.  
   - Mutating routes require `Idempotency-Key`; store hash with 7-day TTL; replay identical requests; 409 on hash mismatch.
3. gRPC owner_api  
   - Protos live in `mesh/contracts/proto/<service>/v1/*.proto` and match DB-02 access modes.  
   - Server exposes owner_api; clients exist for declared DBR/event_projection/replica_view dependencies.  
   - Prefer owner_api over direct DB reads.
4. Events  
   - Every provided/consumed canonical event has schema at `mesh/contracts/events/<event>.json` with producer, class, and partition key from `viralForge/04-services.md`.  
   - Envelope includes `event_id,event_type,occurred_at,source_service,trace_id,schema_version,partition_key_path,partition_key,data`; enforce `partition_key_path = data.<registry_partition_key>`.  
   - Domain-class emits: transactional outbox + DLQ + 7-day dedup. `analytics_only`: best-effort dedup, no DLQ/outbox. `ops`: publish to `platform.audit-events`.
5. Data ownership & access  
   - Writes stay in owned tables (DB-01). Cross-service reads follow declared mode (owner_api/event_projection/replica_view) from DB-02; no direct cross-service writes.
6. Contract generation & wiring  
   - Update OpenAPI/proto/event schemas, regenerate stubs/clients/servers, and wire through ports/adapters. Keep domain independent of adapters.

## Regenerate & validate
Run from repo root unless noted:
- `bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check`
- `bash mesh/scripts/validate-mesh-structure.sh --root-path mesh`
- `bash mesh/scripts/gate-mesh-events-contracts.sh --root-path mesh`
- `bash mesh/scripts/contracts-buf-lint.sh --root-path mesh`
- `bash mesh/scripts/contracts-buf-breaking.sh --root-path mesh`
- `bash mesh/scripts/contracts-buf-generate-check.sh --root-path mesh`
- `bash mesh/scripts/run-mesh-gates.sh`
- If canonical specs changed: `cd viralForge; ./gates/run-all-gates.ps1`

## Outputs to report
- Violations fixed/found by category: REST, gRPC owner_api, events, ownership, gates.
- File paths touched and regenerated artifacts.
- Remaining gaps or assumptions.
- Command results (pass/fail) and outstanding failures.
