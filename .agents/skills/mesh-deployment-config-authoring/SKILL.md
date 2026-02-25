---
name: mesh-deployment-config-authoring
description: Author and update deployment artifacts for mesh services, including Dockerfile, docker-compose, and Kubernetes manifests, and create a Kubernetes markdown guide explaining the purpose of each manifest file.
---

# Mesh Deployment Config Authoring

Use this skill when requests involve deployment configuration files for a mesh service.

## 1) Load deployment context first
- Read:
  - target service `README.md`
  - target service `go.mod`
  - target service `configs/default.yaml`
  - target service `cmd/api/main.go` and `cmd/worker/main.go`
  - target service existing deployment files:
    - `Dockerfile`
    - `deploy/compose/service.compose.yaml`
    - `deploy/k8s/*.yaml`
  - `mesh/docs/local-dev.md`
  - `mesh/services/services-index.yaml`
- Reference:
  - `references/k8s-purpose-doc-template.md`

## 2) Author Dockerfile correctly
- Match Go version with `go.mod`.
- Build required binaries (API and worker when both exist).
- Ensure runtime command strategy is explicit:
  - API-only container, worker-only container, or combined dual-process container.
- Use deterministic build steps and minimal runtime assumptions.
- Keep environment and port behavior aligned with service config.

## 3) Author docker-compose config
- Update `deploy/compose/service.compose.yaml` to include:
  - build context/dockerfile
  - required env vars and defaults
  - port mappings (HTTP/gRPC if exposed)
  - dependencies and networks as needed
- If API and worker should run separately, define separate compose services.
- If they run together by design, document that explicitly.

## 4) Author Kubernetes manifests
- Maintain/update service manifests under `deploy/k8s/`:
  - `configmap.yaml`
  - `deployment.yaml`
  - `service.yaml`
  - `hpa.yaml`
  - `pdb.yaml`
  - plus optional ingress/secret/serviceaccount/role manifests when required.
- Ensure labels/selectors are consistent across manifests.
- Ensure probes, resources, and ports align with real runtime behavior.
- Keep names RFC1123-compatible and service-scoped.

## 5) Required Kubernetes documentation file
- Create or update `deploy/k8s/README.md`.
- This file must describe what each Kubernetes manifest does, at minimum:
  - purpose of each file
  - key fields to review (`metadata`, selectors, ports, probes, resources, scaling)
  - operational effect of changing each file
  - any API/worker process model assumptions
- Keep this guide aligned with actual manifest contents.

## 6) Validate before finishing
- Service checks:
```bash
cd mesh/services/<cluster>/<service>
go test ./...
```
- Mesh checks:
```bash
bash mesh/scripts/generate-mesh-index.sh --root-path mesh --check
bash mesh/scripts/validate-mesh-structure.sh --root-path mesh
bash mesh/scripts/run-mesh-gates.sh
```
- Deployment lint/sanity (when tools are available):
  - `docker compose -f deploy/compose/service.compose.yaml config`
  - `kubectl apply --dry-run=client -f deploy/k8s`

## Non-negotiables
- Do not change runtime behavior unintentionally while editing deployment artifacts.
- Do not add hidden dependencies not declared in configs/manifests.
- Keep deployment docs and files synchronized in the same change.

## Output expectations
- List all deployment files created/updated.
- Include whether API/worker are split or combined and why.
- Confirm `deploy/k8s/README.md` was added/updated with per-file purpose notes.
- Report validation commands run and their status.
