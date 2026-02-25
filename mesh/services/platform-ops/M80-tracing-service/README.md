# M80-Tracing-Service

## Module Metadata
- Module ID: M80
- Canonical Name: M80-Tracing-Service
- Runtime Cluster: platform-ops
- Category: Operational & Infrastructure
- Architecture: microservice

## Primary Responsibility
Accept spans in OTLP, Zipkin JSON, or Jaeger Thrift formats with gRPC and HTTP ingestion.

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
- Follow canonical contracts from viralForge/specs/M80-*.md.
