# M97-Team-Service

## Module Metadata
- Module ID: M97
- Canonical Name: M97-Team-Service
- Runtime Cluster: core-platform
- Category: Internal Admin & Operations
- Architecture: microservice

## Primary Responsibility
The system must allow creators to create one team per account or per storefront, with immutable team identity.

## Dependency Snapshot
### DBR Dependencies
- none

### Event Dependencies
- none

### Event Provides
- team.created
- team.invite.accepted
- team.invite.sent
- team.member.added
- team.member.removed
- team.role.changed

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M97-*.md.
