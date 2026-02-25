# M89-Affiliate-Service

## Module Metadata
- Module ID: M89
- Canonical Name: M89-Affiliate-Service
- Runtime Cluster: integrations
- Category: Commerce & Growth
- Architecture: microservice

## Primary Responsibility
Users can generate unique referral links with unguessable tokens.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- affiliate.attribution.created
- affiliate.click.tracked

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M89-*.md.
