# M19-Storage-Lifecycle-Management

## Module Metadata
- Module ID: M19
- Canonical Name: M19-Storage-Lifecycle-Management
- Runtime Cluster: platform-ops
- Category: Operational & Infrastructure
- Architecture: microservice

## Primary Responsibility
System must automatically delete raw footage files 30 days after campaign closure.

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
- Follow canonical contracts from viralForge/specs/M19-*.md.
