# M41-Reward-Engine

## Module Metadata
- Module ID: M41
- Canonical Name: M41-Reward-Engine
- Runtime Cluster: financial-rails
- Category: Financials & Economy
- Architecture: microservice

## Primary Responsibility
System must calculate creator earnings as (views / 1000) " rate_per_1k. Use DECIMAL(12,4) precision to avoid floating-point errors. Support tiered rates and campaign-specific multipliers.

## Dependency Snapshot
### DBR Dependencies
- M01-Authentication-Service
- M04-Campaign-Service
- M08-Voting-Engine
- M11-Distribution-Tracking-Service
- M26-Submission-Service

### Event Dependencies
- submission.auto_approved
- submission.cancelled
- submission.verified
- submission.view_locked
- tracking.metrics.updated

### Event Provides
- reward.calculated
- reward.payout_eligible

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M41-*.md.
