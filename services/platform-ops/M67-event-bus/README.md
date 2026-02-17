# M67-Event-Bus

## Module Metadata
- Module ID: M67
- Canonical Name: M67-Event-Bus
- Runtime Cluster: platform-ops
- Category: Operational & Infrastructure
- Architecture: microservice

## Primary Responsibility
The system must allow any service to publish events to Kafka topics using JSON or Avro format.

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
- Follow canonical contracts from viralForge/specs/M67-*.md.
