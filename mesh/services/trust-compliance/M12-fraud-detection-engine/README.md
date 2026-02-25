# M12-Fraud-Detection-Engine

## Module Metadata
- Module ID: M12
- Canonical Name: M12-Fraud-Detection-Engine
- Runtime Cluster: trust-compliance
- Category: Moderation & Compliance
- Architecture: microservice

## Primary Responsibility
See canonical service specification.

## Dependency Snapshot
### DBR Dependencies
- M08-Voting-Engine
- M11-Distribution-Tracking-Service
- M26-Submission-Service
- M48-Reputation-Service

### Event Dependencies
- tracking.metrics.updated
- vote.created

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M12-*.md.
