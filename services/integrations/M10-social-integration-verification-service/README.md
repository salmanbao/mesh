# M10-Social-Integration-Verification-Service

## Module Metadata
- Module ID: M10
- Canonical Name: M10-Social-Integration-Verification-Service
- Runtime Cluster: integrations
- Category: Distribution & Tracking
- Architecture: microservice

## Primary Responsibility
System must support OAuth 2.0 flows with TikTok, Instagram Graph, and YouTube APIs.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- social.account.connected
- social.compliance.violation
- social.followers_synced
- social.post.validated
- social.status_changed

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M10-*.md.
