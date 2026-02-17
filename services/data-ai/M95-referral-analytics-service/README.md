# M95-Referral-Analytics-Service

## Module Metadata
- Module ID: M95
- Canonical Name: M95-Referral-Analytics-Service
- Runtime Cluster: data-ai
- Category: Analytics & Reporting
- Architecture: microservice

## Primary Responsibility
Consume Kafka topics affiliate.click.tracked, affiliate.attribution.created, transaction.succeeded, user.registered; batch 100 events; max 5-minute latency; deduplicate by event_id.

## Dependency Snapshot
### DBR Dependencies
- M89-Affiliate-Service

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M95-*.md.
