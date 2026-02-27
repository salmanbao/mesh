# M55-Dashboard-Service

## Module Metadata
- Module ID: M55
- Canonical Name: M55-Dashboard-Service
- Runtime Cluster: data-ai
- Category: Analytics & Reporting
- Architecture: microservice

## Primary Responsibility
Serve role-based dashboard views and persist dashboard personalization (layouts, custom views, preferences) while orchestrating upstream owner APIs.

## Dependency Snapshot
### DBR Dependencies
- M02-Profile-Service (owner_api)
- M05-Billing-Service (owner_api)
- M09-Content-Library-Marketplace (owner_api)
- M13-Escrow-Ledger-Service (owner_api)
- M22-Onboarding-Service (owner_api)
- M39-Finance-Service (owner_api)
- M41-Reward-Engine (owner_api)
- M47-Gamification-Service (owner_api)
- M54-Analytics-Service (owner_api)
- M60-Product-Service (owner_api)

### Event Dependencies
- none

### Event Provides
- none (canonical)

### HTTP Provides
- yes

## Owned Tables (Canonical)
- `dashboard_cache_invalidation`
- `dashboard_custom_views`
- `dashboard_layouts`
- `user_preferences`

## API Surface
- `GET /api/v1/dashboard`
- `PUT /api/v1/dashboard/layout` (requires `Idempotency-Key`, TTL 7 days)
- `POST /api/v1/dashboard/views` (requires `Idempotency-Key`, TTL 7 days)
- `POST /api/v1/dashboard/invalidate`

## Operational Notes
- Internal sync dependency calls are modelled as gRPC clients.
- Event dedup store enforced for worker events (TTL 7 days).
- No cross-service direct DB writes; cross-service reads only via declared owner_api ports.
