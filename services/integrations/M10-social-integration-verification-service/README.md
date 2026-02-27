# M10-Social-Integration-Verification-Service

Mesh implementation of the social integration + verification microservice.

## Canonical dependencies
- Provides: `http`, `social.account.connected`, `social.compliance.violation`, `social.followers_synced`, `social.post.validated`, `social.status_changed`
- Depends on: none (no DBR/event deps declared)

## Runtime surfaces
- Public edge (REST): `/v1/social/*`
- Internal sync: gRPC health server (placeholder for future business proto)
- Async: canonical domain events via outbox relay

## Notes
- In-memory repositories/adapters are used for mesh implementation scaffolding and tests.
- Idempotency and event dedup TTL defaults are 7 days.
