# M91-License-Service

## Module Metadata
- Module ID: M91
- Canonical Name: M91-License-Service
- Runtime Cluster: trust-compliance
- Category: Compliance & Data Governance
- Architecture: microservice

## Primary Responsibility
Generate cryptographically strong license keys in format XXXXX-XXXXX-XXXXX-XXXXX (25 chars total) using CSPRNG; keys must be unguessable.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M91-*.md.
