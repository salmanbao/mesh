# M45-Community-Service

Community integrations and access grant management service in the `integrations` cluster.

## Contract Snapshot

- Architecture: `microservice`
- Public interface: REST (`/api/v1/community/*`, `/api/v1/admin/*`)
- Internal sync interface: gRPC (health server wired; business proto pending)
- Canonical async dependencies (consumed): none declared in `viralForge/specs/dependencies.yaml`
- Canonical async provided events: none declared in `viralForge/specs/dependencies.yaml`

## Ownership and Data Access

- Owned canonical tables:
  - `community_audit_log`
  - `community_grants`
  - `community_health_checks`
  - `community_integrations`
  - `product_community_mappings`
- Cross-service direct DB writes: forbidden
- Cross-service reads: no declared DBR dependencies for M45

## Reliability and Semantics

- Mutating API endpoints require idempotency key (TTL 7 days)
- Event dedup repository present (TTL 7 days) for canonical event handler support
- Canonical event ingestion disabled (no canonical event deps declared for M45)
