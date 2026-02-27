# M96-Referral-Fraud-Detection-Service

## Module Metadata
- Module ID: M96
- Canonical Name: M96-Referral-Fraud-Detection-Service
- Runtime Cluster: trust-compliance
- Architecture: microservice

## Primary Responsibility
Score referral events for fraud risk, store fraud decisions/audit trails/disputes, and provide admin fraud metrics.

## Dependency Snapshot
### DBR Dependencies (owner_api)
- M89-Affiliate-Service

### Event Dependencies
- affiliate.attribution.created
- affiliate.click.tracked
- transaction.succeeded
- user.registered

### Event Provides
- none (canonical `dependencies.yaml` declares none)

### HTTP Provides
- yes

## Implementation Notes
- Internal synchronous calls: gRPC owner-api stub to M89.
- Public edge: REST endpoints under `/v1/referral-fraud/*`.
- Async: consumes canonical events only; module-internal emitted events described in spec are intentionally not emitted as cross-service contracts.
