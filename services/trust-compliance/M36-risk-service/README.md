# M36-Risk-Service

## Module Metadata
- Module ID: M36
- Canonical Name: M36-Risk-Service
- Runtime Cluster: trust-compliance
- Category: Moderation & Compliance
- Architecture: microservice

## Primary Responsibility
System must assign risk_score (0.0 1.0) to sellers based on weighted factors. 

## Dependency Snapshot
### DBR Dependencies
- M01-Authentication-Service
- M02-Profile-Service
- M12-Fraud-Detection-Engine
- M35-Moderation-Service
- M44-Resolution-Center
- M48-Reputation-Service

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M36-*.md.
