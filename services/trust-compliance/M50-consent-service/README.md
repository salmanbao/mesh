# M50-Consent-Service

## Module Metadata
- Module ID: M50
- Canonical Name: M50-Consent-Service
- Runtime Cluster: trust-compliance
- Category: Compliance & Data Governance
- Architecture: microservice

## Primary Responsibility
See canonical service specification.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- user.deleted

### Event Provides
- consent.deletion_requested
- consent.updated
- consent.withdrawn

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M50-*.md.
