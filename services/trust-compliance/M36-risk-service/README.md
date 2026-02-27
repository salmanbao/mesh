# M36-Risk-Service

## Module Metadata
- Module ID: M36
- Canonical Name: M36-Risk-Service
- Runtime Cluster: trust-compliance
- Architecture: microservice

## Primary Responsibility
Compute seller risk dashboards, track dispute logs/evidence, and process chargeback webhooks into risk/escrow/debt records.

## Dependency Snapshot
### DBR Dependencies (owner_api)
- M01-Authentication-Service
- M02-Profile-Service
- M12-Fraud-Detection-Engine
- M35-Moderation-Service
- M44-Resolution-Center
- M48-Reputation-Service

### Event Dependencies
- none (canonical `dependencies.yaml` currently declares none)

### Event Provides
- none (canonical `dependencies.yaml` currently declares none)

### HTTP Provides
- yes

## Implementation Notes
- Internal synchronous calls: gRPC client ports (stubbed in-memory adapters in this mesh implementation).
- Public edge: REST (`/api/v1/seller/risk-dashboard`, dispute/evidence endpoints, chargeback webhook ingress).
- Async path accepts canonical envelope validation only; no canonical event processing is implemented because M36 has no canonical event deps/provides declared.
