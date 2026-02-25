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

## Current Runtime Status
- Foundation implemented: bootstrap, config, migrations, PostgreSQL repositories, Redis caches.
- Implemented endpoint slices:
  - `GET /swagger/`
  - `GET /swagger/openapi.yaml`
  - `POST /auth/v1/register`
  - `POST /auth/v1/login`
  - `POST /auth/v1/2fa/verify`
  - `POST /auth/v1/2fa/setup`
  - `POST /auth/v1/password/reset-request`
  - `POST /auth/v1/password/reset`
  - `POST /auth/v1/email/verify-request`
  - `POST /auth/v1/email/verify`
  - `POST /auth/v1/refresh`
  - `POST /auth/v1/logout`
  - `GET /auth/v1/sessions`
  - `DELETE /auth/v1/sessions/{session_id}`
  - `DELETE /auth/v1/sessions`
  - `GET /auth/v1/login-history`
  - `GET /auth/v1/oidc/authorize`
  - `GET /auth/v1/oidc/callback`
  - `POST /auth/v1/oidc/link`
  - `DELETE /auth/v1/oidc/link/{provider}`
- OIDC flow now uses real provider discovery, token exchange, and JWKS-based `id_token` validation.

## Internal gRPC Surface
- Service: `viralforge.auth.v1.AuthInternalService`
- Methods:
  - `ValidateToken`
  - `GetPublicKeys`
- Proto contract: `mesh/contracts/proto/auth/v1/auth_internal.proto`

## Data Ownership
- Canonical owner tables:
  - `users`, `roles`, `sessions`, `login_attempts`, `oauth_connections`, `oauth_tokens`
  - `email_verification_tokens`, `password_reset_tokens`, `totp_secrets`, `backup_codes`, `two_factor_methods`
- Operational tables:
  - `auth_outbox`
  - `auth_idempotency`

## Local Development
1. Start infra (Postgres + Redis) using mesh compose.
2. From this directory run:
   - `make tidy`
   - `make run-api`
3. Worker process:
   - `make run-worker`

## Decision Log
- See `docs/implementation-decisions.md` for locked schema and boundary decisions.
