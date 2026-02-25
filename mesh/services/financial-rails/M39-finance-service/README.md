# M39-Finance-Service

## Module Metadata
- Module ID: M39
- Canonical Name: M39-Finance-Service
- Runtime Cluster: financial-rails
- Category: Financials & Economy
- Architecture: microservice

## Primary Responsibility
System must accept payments via Stripe, PayPal, and crypto (MoonPay) with webhook confirmation for success/failure. PCI-DSS compliant (no card data stored).

## Dependency Snapshot
### DBR Dependencies
- M01-Authentication-Service
- M04-Campaign-Service
- M09-Content-Library-Marketplace
- M13-Escrow-Ledger-Service
- M15-Platform-Fee-Engine
- M60-Product-Service

### Event Dependencies
- none

### Event Provides
- transaction.failed
- transaction.refunded
- transaction.succeeded

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M39-*.md.
