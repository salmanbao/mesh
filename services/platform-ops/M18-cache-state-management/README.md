# M18-Cache-State-Management

## Module Metadata
- Module ID: M18
- Canonical Name: M18-Cache-State-Management
- Runtime Cluster: platform-ops
- Category: Operational & Infrastructure
- Architecture: microservice

## Primary Responsibility
System must cache campaign leaderboards (Redis) to reduce DB load.

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
- Follow canonical contracts from viralForge/specs/M18-*.md.
