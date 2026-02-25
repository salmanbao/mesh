# Kubernetes Manifest Purpose Template

Use this structure for `deploy/k8s/README.md`.

## Overview
- Service name:
- Runtime model: API only / worker only / API+worker
- Exposed ports:
- Dependencies (DB/cache/broker):

## Manifest Files

### `configmap.yaml`
- Purpose:
- Key settings:
- Operational impact when changed:

### `deployment.yaml`
- Purpose:
- Pods/containers started:
- Probes and resource policy:
- Operational impact when changed:

### `service.yaml`
- Purpose:
- Selected pods and exposed ports:
- Operational impact when changed:

### `hpa.yaml`
- Purpose:
- Scaling signals and min/max replicas:
- Operational impact when changed:

### `pdb.yaml`
- Purpose:
- Disruption/safety guarantees:
- Operational impact when changed:

## Optional Manifests (if present)
- `ingress.yaml`: external traffic routing and TLS policy.
- `secret.yaml`: sensitive runtime configuration contract.
- `serviceaccount.yaml` / `role*.yaml`: workload permissions.

## Change Safety Checklist
- Labels/selectors consistent across manifests.
- Probes align with actual health/readiness endpoints.
- Port names and numbers match service/container ports.
- Resource requests/limits reflect runtime behavior.
- API/worker model documented and consistent with deployment.
