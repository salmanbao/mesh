# M30-Social-Integration-Service

## Module Metadata
- Module ID: M30
- Canonical Name: M30-Social-Integration-Service
- Runtime Cluster: integrations
- Category: Distribution & Tracking
- Architecture: microservice

## Primary Responsibility
See canonical service specification.

## Dependency Snapshot
### DBR Dependencies
- M10-Social-Integration-Verification-Service

### Event Dependencies
- social.account.connected
- social.compliance.violation
- social.followers_synced
- social.post.validated
- social.status_changed

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M30-*.md.
