#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=mesh/scripts/libmesh.sh
source "$SCRIPT_DIR/libmesh.sh"

ROOT_PATH="mesh"
ARCHITECTURE_MAP_PATH="viralForge/specs/service-architecture-map.yaml"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --root-path) ROOT_PATH="$2"; shift 2 ;;
    --architecture-map-path) ARCHITECTURE_MAP_PATH="$2"; shift 2 ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ ! -d "$ROOT_PATH" ]]; then
  echo "Mesh root missing: $ROOT_PATH" >&2
  exit 1
fi
ROOT_ABS="$(cd "$ROOT_PATH" && pwd)"

load_microservices "$ARCHITECTURE_MAP_PATH"

errors=()
if [[ "${#SERVICES[@]}" -ne 49 ]]; then
  errors+=("Expected 49 microservices from architecture map, got ${#SERVICES[@]}")
fi

required_root_paths=(
  "README.md"
  "go.work"
  "docs/architecture-principles.md"
  "docs/service-lifecycle.md"
  "docs/local-dev.md"
  "docs/dependency-load-order.md"
  "platform/version.yaml"
  "platform/go.mod"
  "contracts/proto/README.md"
  "contracts/openapi/README.md"
  "contracts/events/README.md"
  "contracts/schemas/README.md"
  "environments/compose/docker-compose.base.yaml"
  "environments/compose/docker-compose.services.yaml"
  "services/services-index.yaml"
)

rel=""
for rel in "${required_root_paths[@]}"; do
  if [[ ! -e "$ROOT_PATH/$rel" ]]; then
    errors+=("Missing required mesh file: $rel")
  fi
done

cluster_dirs=(core-platform integrations trust-compliance data-ai financial-rails platform-ops)
cluster=""
for cluster in "${cluster_dirs[@]}"; do
  if [[ ! -d "$ROOT_PATH/services/$cluster" ]]; then
    errors+=("Missing service cluster directory: services/$cluster")
  fi
done

svc=""
for svc in "${SERVICES[@]}"; do
  id="${SERVICE_ID[$svc]}"
  dir_name="$(directory_name "$id" "${SERVICE_NAME[$svc]}")"
  mapfile -t matches < <(find "$ROOT_PATH/services" -type d -name "$dir_name")

  if [[ "${#matches[@]}" -eq 0 ]]; then
    errors+=("Missing service directory for $svc: expected $dir_name")
    continue
  fi
  if [[ "${#matches[@]}" -gt 1 ]]; then
    paths_joined="$(printf '%s, ' "${matches[@]}")"
    paths_joined="${paths_joined%, }"
    errors+=("Service directory appears multiple times for $svc: $paths_joined")
    continue
  fi

  service_root="${matches[0]}"
  required_service_files=(
    "README.md"
    "go.mod"
    "cmd/api/main.go"
    "cmd/worker/main.go"
    "internal/app/bootstrap/bootstrap.go"
    "configs/default.yaml"
    "deploy/k8s/deployment.yaml"
    "deploy/k8s/service.yaml"
    "deploy/k8s/configmap.yaml"
    "deploy/k8s/hpa.yaml"
    "deploy/k8s/pdb.yaml"
    "deploy/compose/service.compose.yaml"
    ".golangci.yml"
    "Makefile"
    "tests/unit/.gitkeep"
    "tests/integration/.gitkeep"
    "tests/contract/.gitkeep"
  )

  rf=""
  for rf in "${required_service_files[@]}"; do
    if [[ ! -e "$service_root/$rf" ]]; then
      errors+=("Missing service file for $svc: $rf")
    fi
  done

  go_mod="$service_root/go.mod"
  if [[ -f "$go_mod" ]]; then
    first_line="$(head -n1 "$go_mod" | tr -d '\r')"
    cluster_name="$(basename "$(dirname "$service_root")")"
    expected="module github.com/viralforge/mesh/services/$cluster_name/$dir_name"
    if [[ "$first_line" != "$expected" ]]; then
      errors+=("Invalid go.mod module path for $svc: got '$first_line', expected '$expected'")
    fi
  fi
done

if ((${#errors[@]} > 0)); then
  echo "validate-mesh-structure: FAILED"
  err=""
  for err in "${errors[@]}"; do
    echo "$err" >&2
  done
  exit 1
fi

echo "validate-mesh-structure: passed (${#SERVICES[@]} services validated)"
