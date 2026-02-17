# M58-Content-Recommendation-Engine

## Module Metadata
- Module ID: M58
- Canonical Name: M58-Content-Recommendation-Engine
- Runtime Cluster: data-ai
- Category: AI & Automation
- Architecture: microservice

## Primary Responsibility
System must use "users who clipped X also liked Y" logic to suggest campaigns. Support both user-user and item-item similarity with tunable weight.  

## Dependency Snapshot
### DBR Dependencies
- M23-Campaign-Discovery-Service

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M58-*.md.
