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

## Current Runtime Status
- Foundation implemented: bootstrap, config, migrations, PostgreSQL repositories, Redis cache, outbox worker, Kafka adapters (fallback to logging/noop when Kafka env not configured).
- Swagger docs exposed:
  - `GET /swagger/`
  - `GET /swagger/openapi.yaml`
- Implemented endpoint slices:
  - `GET /v1/profiles/me`
  - `GET /v1/profiles/{username}`
  - `PUT /v1/profiles/me`
  - `PUT /v1/profiles/me/username`
  - `GET /v1/profiles/username-availability`
  - `POST /v1/profiles/me/avatar`
  - `POST /v1/profiles/me/social-links`
  - `DELETE /v1/profiles/me/social-links/{platform}`
  - `POST /v1/profiles/me/payout-methods`
  - `PUT /v1/profiles/me/payout-methods/{method_type}`
  - `POST /v1/profiles/me/kyc/documents`
  - `GET /v1/profiles/me/kyc/status`
  - `GET /v1/admin/profiles`
  - `GET /v1/admin/kyc/queue`
  - `POST /v1/admin/kyc/{user_id}/approve`
  - `POST /v1/admin/kyc/{user_id}/reject`

## Internal gRPC Surface
- Service: `viralforge.profile.v1.ProfileInternalService`
- Methods:
  - `GetProfile`
  - `BatchGetProfiles`
- Proto contract: `mesh/contracts/proto/profile/v1/profile_internal.proto`

## Event Contracts
- Consumes:
  - `user.registered`
  - `user.deleted`
- Emits:
  - `user.profile_updated`
- Outbox-backed publish path with partition invariant:
  - `partition_key_path=data.user_id`
  - `partition_key={user_id}`

## Data Ownership
- Canonical owner tables:
  - `profiles`, `social_links`, `payout_methods`, `kyc_documents`, `username_history`, `profile_stats`, `reserved_usernames`
- Operational tables:
  - `profile_outbox`
  - `profile_idempotency`
  - `profile_event_dedup`
