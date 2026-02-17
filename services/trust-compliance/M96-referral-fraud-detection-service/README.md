# M96-Referral-Fraud-Detection-Service

## Module Metadata
- Module ID: M96
- Canonical Name: M96-Referral-Fraud-Detection-Service
- Runtime Cluster: trust-compliance
- Category: Moderation & Compliance
- Architecture: microservice

## Primary Responsibility
The service must assign a `risk_score` (0 1.0) to each referral event using an ML model and block events above a configurable threshold.

## Dependency Snapshot
### DBR Dependencies
- M89-Affiliate-Service

### Event Dependencies
- affiliate.attribution.created
- affiliate.click.tracked
- transaction.succeeded
- user.registered

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M96-*.md.
