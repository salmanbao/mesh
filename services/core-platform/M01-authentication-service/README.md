# M01-Authentication-Service

## Module Metadata
- Module ID: M01
- Canonical Name: M01-Authentication-Service
- Runtime Cluster: core-platform
- Category: Core Platform & Foundation
- Architecture: microservice

## Primary Responsibility
System must support user registration using Email/Password and OIDC (Google).

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- auth.2fa.required
- user.deleted
- user.registered

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M01-*.md.
