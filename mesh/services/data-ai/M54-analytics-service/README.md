# M54-Analytics-Service

## Module Metadata
- Module ID: M54
- Canonical Name: M54-Analytics-Service
- Runtime Cluster: data-ai
- Category: Analytics & Reporting
- Architecture: microservice

## Primary Responsibility
System must consume events from Kafka (submission.created, submission.approved, payout.paid, reward.calculated, campaign.launched, user.registered, transaction.succeeded).  

## Dependency Snapshot
### DBR Dependencies
- M08-Voting-Engine
- M10-Social-Integration-Verification-Service
- M11-Distribution-Tracking-Service
- M26-Submission-Service
- M39-Finance-Service

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M54-*.md.
