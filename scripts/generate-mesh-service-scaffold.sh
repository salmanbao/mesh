#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=mesh/scripts/libmesh.sh
source "$SCRIPT_DIR/libmesh.sh"

ROOT_PATH="mesh"
ARCHITECTURE_MAP_PATH="viralForge/specs/service-architecture-map.yaml"
DEPENDENCIES_PATH="viralForge/specs/dependencies.yaml"
DEPLOYMENT_PROFILE_PATH="viralForge/specs/service-deployment-profile.md"
SPECS_DIR="viralForge/specs"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --root-path) ROOT_PATH="$2"; shift 2 ;;
    --architecture-map-path) ARCHITECTURE_MAP_PATH="$2"; shift 2 ;;
    --dependencies-path) DEPENDENCIES_PATH="$2"; shift 2 ;;
    --deployment-profile-path) DEPLOYMENT_PROFILE_PATH="$2"; shift 2 ;;
    --specs-dir) SPECS_DIR="$2"; shift 2 ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

mkdir -p "$ROOT_PATH"
ROOT_ABS="$(cd "$ROOT_PATH" && pwd)"

load_microservices "$ARCHITECTURE_MAP_PATH"
load_dependencies "$DEPENDENCIES_PATH"
load_categories_from_profile "$DEPLOYMENT_PROFILE_PATH"
load_suggested_clusters_from_profile "$DEPLOYMENT_PROFILE_PATH"
build_clustered_maps

cat > "$ROOT_PATH/README.md" <<'EOF'
# Mesh Microservices Workspace

`mesh` hosts all services classified as `architecture: microservice` in `viralForge/specs/service-architecture-map.yaml`.

## Scope
- Service model: one Go module per microservice.
- Runtime model: Kubernetes-first manifests with Docker Compose local parity.
- Interface model: gRPC for internal service-to-service calls, REST for external/public APIs.
- Shared technical primitives: versioned libraries under `mesh/platform`.

## Source Of Truth
- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/service-deployment-profile.md`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `viralForge/specs/dependencies.yaml`

## Non-Goals
- Implementing monolith services (those remain in `solomon`).
- Overriding canonical contracts or ownership boundaries defined in specs.
EOF

go_work_entries=("./platform")
svc=""
for svc in "${SERVICES[@]}"; do
  go_work_entries+=("./services/${SVC_CLUSTER[$svc]}/${SVC_DIR[$svc]}")
done
{
  echo "go 1.22.0"
  echo
  echo "use ("
  printf '%s\n' "${go_work_entries[@]}" | LC_ALL=C sort -u | while IFS= read -r entry; do
    [[ -z "$entry" ]] && continue
    echo "    $entry"
  done
  echo ")"
} > "$ROOT_PATH/go.work"

cat > "$ROOT_PATH/.env.example" <<'EOF'
# Mesh local environment
POSTGRES_URL=postgresql://postgres:postgres@localhost:5432/mesh
REDIS_URL=redis://localhost:6379
KAFKA_BROKERS=localhost:9092
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
EOF

mkdir -p "$ROOT_PATH/platform"
cat > "$ROOT_PATH/platform/go.mod" <<'EOF'
module github.com/viralforge/mesh/platform

go 1.22.0
EOF
cat > "$ROOT_PATH/platform/version.yaml" <<'EOF'
version: 0.1.0
compatibility: semver
EOF
cat > "$ROOT_PATH/platform/README.md" <<'EOF'
# Mesh Shared Platform Libraries

This module contains reusable technical primitives for microservices:
- config
- logging
- observability
- grpc
- http
- messaging
- security
- resiliency

Business/domain logic is not allowed in this module.
EOF

platform_packages=(config logging observability grpc http messaging security resiliency)
for pkg in "${platform_packages[@]}"; do
  mkdir -p "$ROOT_PATH/platform/$pkg"
  cat > "$ROOT_PATH/platform/$pkg/$pkg.go" <<EOF
package $pkg

// Package $pkg contains shared technical primitives for mesh services.
EOF
done

mkdir -p "$ROOT_PATH/contracts/proto" "$ROOT_PATH/contracts/openapi" "$ROOT_PATH/contracts/events" "$ROOT_PATH/contracts/schemas"
cat > "$ROOT_PATH/contracts/README.md" <<'EOF'
# Mesh Contracts

Contract-first artifacts for all microservices:
- `proto`: internal gRPC contracts
- `openapi`: external REST contracts
- `events`: event envelopes and schemas
- `schemas`: shared payload schemas

All contract changes must preserve backward compatibility unless versioned explicitly.
EOF
echo "# gRPC Contracts" > "$ROOT_PATH/contracts/proto/README.md"
echo "# OpenAPI Contracts" > "$ROOT_PATH/contracts/openapi/README.md"
echo "# Event Contracts" > "$ROOT_PATH/contracts/events/README.md"
echo "# Shared Schemas" > "$ROOT_PATH/contracts/schemas/README.md"

mkdir -p "$ROOT_PATH/docs"
cat > "$ROOT_PATH/docs/architecture-principles.md" <<'EOF'
# Mesh Architecture Principles

## Boundary Rules
- `mesh` contains only services marked `architecture: microservice`.
- `solomon` remains the monolith runtime; no duplication of service implementation.
- Cross-service writes are prohibited.

## Communication
- Internal synchronous communication: gRPC.
- Public/external interfaces: REST.
- Asynchronous integration: canonical events from `dependencies.yaml`.

## Shared Code
- Shared code is technical only and versioned in `mesh/platform`.
- Domain/business logic must remain service-local.
EOF

cat > "$ROOT_PATH/docs/service-lifecycle.md" <<'EOF'
# Mesh Service Lifecycle

## Add a New Microservice
1. Mark service as `microservice` in `service-architecture-map.yaml`.
2. Update dependencies in `dependencies.yaml`.
3. Run `bash scripts/generate-mesh-service-scaffold.sh` and `bash scripts/generate-mesh-index.sh`.
4. Implement contracts and tests.

## Change Contracts
1. Update protobuf/openapi/event schemas.
2. Verify backward compatibility.
3. Update service README and changelog.

## Deprecate a Microservice
1. Mark deprecated in canonical maps.
2. Define successor service.
3. Keep migration notes until full cutover.
EOF

cat > "$ROOT_PATH/docs/local-dev.md" <<'EOF'
# Mesh Local Development

## Prerequisites
- Go 1.22+
- Docker + Docker Compose

## Start Infra
`docker compose -f environments/compose/docker-compose.base.yaml up -d`

## Run One Service
`cd services/<cluster>/<service>` then `go run ./cmd/api`

## Run Validation
`bash scripts/validate-mesh-structure.sh`
`bash scripts/generate-mesh-index.sh --check`
EOF

mkdir -p "$ROOT_PATH/environments/compose" "$ROOT_PATH/environments/k8s/base" "$ROOT_PATH/environments/k8s/overlays"
cat > "$ROOT_PATH/environments/compose/docker-compose.base.yaml" <<'EOF'
name: mesh
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: mesh
    ports: ["5432:5432"]
    networks: [mesh]
  redis:
    image: redis:7
    ports: ["6379:6379"]
    networks: [mesh]
  kafka:
    image: bitnami/kafka:3.7
    environment:
      KAFKA_CFG_NODE_ID: 1
      KAFKA_CFG_PROCESS_ROLES: broker,controller
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CFG_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CFG_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
    ports: ["9092:9092"]
    networks: [mesh]
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.100.0
    ports: ["4317:4317"]
    networks: [mesh]
networks:
  mesh: {}
EOF

{
  echo "services:"
  for svc in "${SERVICES[@]}"; do
    service_key="${SVC_DIR[$svc],,}"
    echo "  ${service_key}:"
    echo "    build:"
    echo "      context: ../../services/${SVC_CLUSTER[$svc]}/${SVC_DIR[$svc]}"
    echo "      dockerfile: Dockerfile"
    echo "    env_file:"
    echo "      - ../../.env.example"
    echo "    networks: [mesh]"
  done
  echo "networks:"
  echo "  mesh: {}"
} > "$ROOT_PATH/environments/compose/docker-compose.services.yaml"

echo "# Kubernetes Base Manifests" > "$ROOT_PATH/environments/k8s/base/README.md"
echo "# Kubernetes Overlays" > "$ROOT_PATH/environments/k8s/overlays/README.md"

mkdir -p "$ROOT_PATH/tooling/service-template" "$ROOT_PATH/tooling/manifests"
cat > "$ROOT_PATH/tooling/service-template/README.md" <<'EOF'
# Service Template

Template material for new mesh microservices.
Use generator scripts to scaffold consistently.
EOF

cluster_dirs=(core-platform integrations trust-compliance data-ai financial-rails platform-ops)
for cluster in "${cluster_dirs[@]}"; do
  mkdir -p "$ROOT_PATH/services/$cluster"
done

for svc in "${SERVICES[@]}"; do
  service_root="$ROOT_PATH/services/${SVC_CLUSTER[$svc]}/${SVC_DIR[$svc]}"
  mkdir -p \
    "$service_root/cmd/api" \
    "$service_root/cmd/worker" \
    "$service_root/internal/app/bootstrap" \
    "$service_root/internal/domain" \
    "$service_root/internal/application" \
    "$service_root/internal/ports" \
    "$service_root/internal/adapters/http" \
    "$service_root/internal/adapters/grpc" \
    "$service_root/internal/adapters/events" \
    "$service_root/internal/adapters/postgres" \
    "$service_root/internal/contracts" \
    "$service_root/configs" \
    "$service_root/deploy/k8s" \
    "$service_root/deploy/compose" \
    "$service_root/tests/unit" \
    "$service_root/tests/integration" \
    "$service_root/tests/contract"

  spec_summary="$(get_spec_summary "$SPECS_DIR" "${SERVICE_ID[$svc]}")"
  dbr_text="$(bullet_or_none "$(map_sorted_unique DEP_DBR "$svc")")"
  event_deps_text="$(bullet_or_none "$(map_sorted_unique DEP_EVENT_DEPS "$svc")")"
  event_provides_text="$(bullet_or_none "$(map_sorted_unique DEP_EVENT_PROVIDES "$svc")")"
  http_provides="no"
  if [[ "${DEP_HTTP[$svc]-0}" == "1" ]]; then
    http_provides="yes"
  fi

  cat > "$service_root/README.md" <<EOF
# $svc

## Module Metadata
- Module ID: ${SERVICE_ID[$svc]}
- Canonical Name: $svc
- Runtime Cluster: ${SVC_CLUSTER[$svc]}
- Category: ${SVC_CATEGORY[$svc]}
- Architecture: microservice

## Primary Responsibility
$spec_summary

## Dependency Snapshot
### DBR Dependencies
$dbr_text

### Event Dependencies
$event_deps_text

### Event Provides
$event_provides_text

### HTTP Provides
- $http_provides

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/${SERVICE_ID[$svc]}-*.md.
EOF

  cat > "$service_root/go.mod" <<EOF
module github.com/viralforge/mesh/services/${SVC_CLUSTER[$svc]}/${SVC_DIR[$svc]}

go 1.22.0
EOF

  cat > "$service_root/cmd/api/main.go" <<EOF
package main

import "fmt"

func main() {
    fmt.Println("$svc API placeholder")
}
EOF

  cat > "$service_root/cmd/worker/main.go" <<EOF
package main

import "fmt"

func main() {
    fmt.Println("$svc worker placeholder")
}
EOF

  cat > "$service_root/internal/app/bootstrap/bootstrap.go" <<'EOF'
package bootstrap

// Build wires runtime dependencies for this service.
func Build() error {
    return nil
}
EOF

  : > "$service_root/internal/domain/.gitkeep"
  : > "$service_root/internal/application/.gitkeep"
  : > "$service_root/internal/ports/.gitkeep"
  : > "$service_root/internal/adapters/http/.gitkeep"
  : > "$service_root/internal/adapters/grpc/.gitkeep"
  : > "$service_root/internal/adapters/events/.gitkeep"
  : > "$service_root/internal/adapters/postgres/.gitkeep"
  : > "$service_root/internal/contracts/.gitkeep"
  : > "$service_root/tests/unit/.gitkeep"
  : > "$service_root/tests/integration/.gitkeep"
  : > "$service_root/tests/contract/.gitkeep"

  cat > "$service_root/configs/default.yaml" <<EOF
service:
  id: $svc
  cluster: ${SVC_CLUSTER[$svc]}
  http_port: 8080
  grpc_port: 9090
dependencies:
  postgres_url: \${POSTGRES_URL}
  redis_url: \${REDIS_URL}
  kafka_brokers: \${KAFKA_BROKERS}
observability:
  otlp_endpoint: \${OTEL_EXPORTER_OTLP_ENDPOINT}
EOF

  cat > "$service_root/Dockerfile" <<'EOF'
FROM golang:1.22-alpine
WORKDIR /app
COPY . .
RUN go build -o /out/api ./cmd/api
CMD ["/out/api"]
EOF

  cat > "$service_root/deploy/k8s/deployment.yaml" <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${SVC_DIR[$svc]}
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ${SVC_DIR[$svc]}
  template:
    metadata:
      labels:
        app: ${SVC_DIR[$svc]}
    spec:
      containers:
        - name: api
          image: ${SVC_DIR[$svc]}:latest
          ports:
            - containerPort: 8080
            - containerPort: 9090
EOF

  cat > "$service_root/deploy/k8s/service.yaml" <<EOF
apiVersion: v1
kind: Service
metadata:
  name: ${SVC_DIR[$svc]}
spec:
  selector:
    app: ${SVC_DIR[$svc]}
  ports:
    - name: http
      port: 80
      targetPort: 8080
    - name: grpc
      port: 9090
      targetPort: 9090
EOF

  cat > "$service_root/deploy/k8s/configmap.yaml" <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ${SVC_DIR[$svc]}-config
data:
  CONFIG_PATH: /app/configs/default.yaml
EOF

  cat > "$service_root/deploy/k8s/hpa.yaml" <<EOF
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ${SVC_DIR[$svc]}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ${SVC_DIR[$svc]}
  minReplicas: 2
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
EOF

  cat > "$service_root/deploy/k8s/pdb.yaml" <<EOF
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: ${SVC_DIR[$svc]}
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: ${SVC_DIR[$svc]}
EOF

  cat > "$service_root/deploy/compose/service.compose.yaml" <<EOF
services:
  ${SVC_DIR[$svc],,}:
    build:
      context: ../../services/${SVC_CLUSTER[$svc]}/${SVC_DIR[$svc]}
      dockerfile: Dockerfile
    env_file:
      - ../../.env.example
    networks: [mesh]
networks:
  mesh: {}
EOF

  cat > "$service_root/.golangci.yml" <<'EOF'
run:
  timeout: 5m
linters:
  enable:
    - govet
    - staticcheck
EOF

  cat > "$service_root/Makefile" <<'EOF'
.PHONY: run-api run-worker test

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./...
EOF
done

echo "generate-mesh-service-scaffold: scaffolded ${#SERVICES[@]} microservices under $ROOT_ABS"
