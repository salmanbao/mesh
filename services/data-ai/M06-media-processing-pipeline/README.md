# M06-Media-Processing-Pipeline

## Module Metadata
- Module ID: M06
- Canonical Name: M06-Media-Processing-Pipeline
- Runtime Cluster: data-ai
- Category: Editorial Workflow
- Architecture: microservice

## Primary Responsibility
System must transcode uploaded videos into standard 1080p and 720p profiles.

## Dependency Snapshot
### DBR Dependencies
- M04-Campaign-Service

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M06-*.md.
