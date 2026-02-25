# M05-Billing-Service

## Module Metadata
- Module ID: M05
- Canonical Name: M05-Billing-Service
- Runtime Cluster: financial-rails
- Category: Financials & Economy
- Architecture: microservice

## Primary Responsibility
Automatically generate itemized invoices upon: user purchase, subscription renewal, creator payout, and payout disbursement. Idempotent processing.

## Dependency Snapshot
### DBR Dependencies
- M01-Authentication-Service
- M09-Content-Library-Marketplace
- M15-Platform-Fee-Engine
- M39-Finance-Service
- M61-Subscription-Service

### Event Dependencies
- payout.failed
- payout.paid

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M05-*.md.
