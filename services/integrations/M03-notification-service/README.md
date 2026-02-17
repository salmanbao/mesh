# M03-Notification-Service

## Module Metadata
- Module ID: M03
- Canonical Name: M03-Notification-Service
- Runtime Cluster: integrations
- Category: Notifications & Alerts
- Architecture: microservice

## Primary Responsibility
List user's notifications with filtering and pagination

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- auth.2fa.required
- campaign.budget_updated
- campaign.created
- campaign.launched
- dispute.created
- payout.failed
- payout.paid
- submission.approved
- submission.rejected
- transaction.failed
- user.registered

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M03-*.md.
