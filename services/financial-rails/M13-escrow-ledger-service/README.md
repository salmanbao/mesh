# M13-Escrow-Ledger-Service

## Module Metadata
- Module ID: M13
- Canonical Name: M13-Escrow-Ledger-Service
- Runtime Cluster: financial-rails
- Category: Financials & Economy
- Architecture: microservice

## Primary Responsibility
See canonical service specification.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- escrow.hold_created
- escrow.hold_fully_released
- escrow.partial_release
- escrow.refund_processed

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M13-*.md.
