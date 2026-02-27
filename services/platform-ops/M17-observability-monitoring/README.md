# M17-Observability-Monitoring

Mesh implementation of the observability/monitoring service with clean layering under `internal/{domain,application,ports,adapters,contracts}`.

## Scope
- Public REST endpoints:
  - `GET /health`
  - `GET /metrics` (Prometheus text format)
- Internal admin REST helpers for mock component state:
  - `GET /api/v1/observability/components`
  - `PUT /api/v1/observability/components/{name}`
- Internal gRPC runtime: health-check protocol only
- No canonical event consume/emit (per canonical dependencies)

## Canonical Alignment
- `dependencies.yaml` declares `provides: [http]` and `depends_on: []` for M17.
- `service-data-ownership-map.yaml` declares no owned canonical tables and no DBR reads.
- Spec also states M17 does not emit or consume Kafka events.

## Implementation Notes
- Health endpoint returns spec-aligned JSON with `database`, `redis`, and `kafka` checks.
- Metrics endpoint exports Prometheus-style counters/histograms from in-memory request instrumentation.
- Event handler path still enforces envelope validation + 7-day dedup and rejects unsupported canonical events.
- Idempotency store (7-day TTL) is applied to the admin component update endpoint.

## Local Run
```bash
go test ./...
go run ./cmd/api
go run ./cmd/worker
```
