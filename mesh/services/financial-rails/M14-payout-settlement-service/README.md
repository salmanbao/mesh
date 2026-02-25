# M14-Payout-Settlement-Service

## Module Metadata
- Module ID: M14
- Canonical Name: M14-Payout-Settlement-Service
- Runtime Cluster: financial-rails
- Category: Financials & Economy
- Architecture: microservice

## Primary Responsibility
See canonical service specification.

## Dependency Snapshot
### DBR Dependencies
- M01-Authentication-Service
- M02-Profile-Service
- M05-Billing-Service
- M13-Escrow-Ledger-Service
- M36-Risk-Service
- M39-Finance-Service
- M41-Reward-Engine

### Event Dependencies
- reward.payout_eligible

### Event Provides
- payout.failed
- payout.paid
- payout.processing

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M14-*.md.
