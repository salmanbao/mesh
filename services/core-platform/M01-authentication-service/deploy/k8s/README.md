# M01 Kubernetes Deployment Files

## Overview
- Service: `M01-Authentication-Service`
- Runtime model: split workloads from one image
  - API pod runs `/out/api`
  - Worker pod runs `/out/worker`
- Exposed ports:
  - HTTP: `8080` (service port `80`)
  - gRPC: `9090`

## File Purposes

### `configmap.yaml`
- Purpose: provides non-sensitive runtime configuration shared by API and worker pods.
- Key fields to review:
  - auth/session policy env vars (`TOKEN_EXPIRY_HOURS`, `FAILED_LOGIN_THRESHOLD`, etc.)
  - outbox worker tuning (`OUTBOX_POLL_SECONDS`, `OUTBOX_BATCH_SIZE`)
  - port config (`HTTP_PORT`, `GRPC_PORT`)
- Operational impact:
  - changing values alters auth behavior, lockout policy, session TTLs, and worker processing cadence.

### `deployment.yaml`
- Purpose: API workload deployment.
- Key fields to review:
  - `replicas`
  - container `command: ["/out/api"]`
  - readiness/liveness probes (`/readyz`, `/healthz`)
  - resource requests/limits
  - labels/selectors (`app.kubernetes.io/*`)
- Operational impact:
  - controls API availability, rollout behavior, and resource usage.

### `worker-deployment.yaml`
- Purpose: background worker workload deployment for outbox/event publishing.
- Key fields to review:
  - `replicas` (default `1`)
  - container `command: ["/out/worker"]`
  - env sources and worker resource limits
- Operational impact:
  - scales async processing independently from API traffic.
  - increasing replicas can raise publish throughput but may increase duplicate-processing risk if handlers are not strictly idempotent.

### `service.yaml`
- Purpose: stable in-cluster endpoint for API pods only.
- Key fields to review:
  - selector includes `app.kubernetes.io/component: api`
  - HTTP and gRPC ports
- Operational impact:
  - routes network traffic only to API pods; worker pods are intentionally excluded.

### `hpa.yaml`
- Purpose: autoscaling policy for API deployment.
- Key fields to review:
  - min/max replicas
  - CPU utilization target
  - `scaleTargetRef` to API deployment
- Operational impact:
  - automatically adjusts API replica count under load.

### `pdb.yaml`
- Purpose: disruption safety for API pods during voluntary disruptions.
- Key fields to review:
  - `minAvailable`
  - selector targeting API pods only
- Operational impact:
  - prevents too many API pods from being evicted simultaneously.

## Process Model Assumptions
- The built image contains both binaries (`/out/api`, `/out/worker`).
- API and worker are deployed as separate Kubernetes Deployments for independent scaling.
- Secrets (DB URL, Redis URL, JWT/OIDC credentials) are provided via `m01-authentication-service-secrets`.
