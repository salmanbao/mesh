# M02-Profile-Service

## Module Metadata
- Module ID: M02
- Canonical Name: M02-Profile-Service
- Runtime Cluster: core-platform
- Category: Core Platform & Foundation
- Architecture: microservice

## Primary Responsibility
System must create profile automatically after user registration via auth-service. Default display_name = email prefix (first part before @). Initialize empty bio, no avatar, no social links, payout_method unset.

## Dependency Snapshot
### DBR Dependencies
- M01-Authentication-Service

### Event Dependencies
- user.deleted
- user.registered

### Event Provides
- user.profile_updated

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M02-*.md.
