# M45-Community-Service

## Module Metadata
- Module ID: M45
- Canonical Name: M45-Community-Service
- Runtime Cluster: integrations
- Category: Community & Engagement
- Architecture: microservice

## Primary Responsibility
System must support automated access to external communities (Discord, Slack, Telegram) and internal chat.  

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
- Follow canonical contracts from viralForge/specs/M45-*.md.
