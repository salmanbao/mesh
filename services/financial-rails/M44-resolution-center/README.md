# M44-Resolution-Center

## Module Metadata
- Module ID: M44
- Canonical Name: M44-Resolution-Center
- Runtime Cluster: financial-rails
- Category: Financials & Economy
- Architecture: microservice

## Primary Responsibility
System must allow users to submit refund requests with justification and supporting evidence.  

## Dependency Snapshot
### DBR Dependencies
- M35-Moderation-Service

### Event Dependencies
- payout.failed
- submission.approved

### Event Provides
- dispute.created
- dispute.resolved
- transaction.refunded

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M44-*.md.
