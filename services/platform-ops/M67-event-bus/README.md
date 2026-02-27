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
- Internal service calls: gRPC (runtime currently exposes health-only gRPC server).
- External/public interfaces: REST.
- Canonical async events: none declared for M67 itself; canonical event handler validates envelope + 7-day dedup then rejects unsupported event types.
- Mutating REST endpoints enforce `Idempotency-Key` with 7-day TTL.
- In-memory repositories model M67-owned metadata tables (`kafka_topics`, `kafka_acls`, `kafka_consumer_offsets`, `dlq_messages`) plus service-local schema/idempotency stores.
