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
if [[ "${#SERVICES[@]}" -eq 0 ]]; then
  errors+=("No microservices were discovered from architecture map: $ARCHITECTURE_MAP_PATH")
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
  "contracts/go.mod"
  "contracts/buf.yaml"
  "contracts/buf.gen.yaml"
  "contracts/proto/README.md"
  "contracts/openapi/README.md"
  "contracts/events/README.md"
  "contracts/schemas/README.md"
  "environments/compose/docker-compose.base.yaml"
  "environments/compose/docker-compose.services.yaml"
  "services/services-index.yaml"
  "tooling/manifests/implemented-services.yaml"
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

is_rfc1123_label() {
  local value="$1"
  [[ "$value" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]]
}

metadata_name() {
  local file="$1"
  awk '
    BEGIN { in_meta = 0 }
    /^[[:space:]]*metadata:[[:space:]]*$/ { in_meta = 1; next }
    in_meta == 1 {
      if ($0 ~ /^[[:space:]]*name:[[:space:]]*/) {
        gsub(/^[[:space:]]*name:[[:space:]]*/, "", $0)
        gsub(/[[:space:]]+$/, "", $0)
        print $0
        exit
      }
      if ($0 ~ /^[^[:space:]]/) { in_meta = 0 }
    }
  ' "$file"
}

is_bootstrap_package_valid() {
  local bootstrap_dir="$1"
  mapfile -t bootstrap_files < <(find "$bootstrap_dir" -maxdepth 1 -type f -name "*.go" | LC_ALL=C sort)

  if [[ "${#bootstrap_files[@]}" -eq 0 ]]; then
    return 1
  fi

  local has_pkg="0"
  local has_entrypoint="0"
  local file=""
  for file in "${bootstrap_files[@]}"; do
    if grep -Eq '^[[:space:]]*package[[:space:]]+bootstrap([[:space:]]|$)' "$file"; then
      has_pkg="1"
    fi
    if grep -Eq '^[[:space:]]*func[[:space:]]+(Build|NewRuntime)[[:space:]]*\(' "$file"; then
      has_entrypoint="1"
    fi
  done

  [[ "$has_pkg" == "1" && "$has_entrypoint" == "1" ]]
}

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
    "internal/app/bootstrap"
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

  bootstrap_dir="$service_root/internal/app/bootstrap"
  if [[ -d "$bootstrap_dir" ]]; then
    if ! is_bootstrap_package_valid "$bootstrap_dir"; then
      errors+=("Invalid bootstrap package for $svc: expected package 'bootstrap' with exported Build() or NewRuntime(...) in internal/app/bootstrap/*.go")
    fi
  fi

  go_mod="$service_root/go.mod"
  if [[ -f "$go_mod" ]]; then
    first_line="$(head -n1 "$go_mod" | tr -d '\r')"
    cluster_name="$(basename "$(dirname "$service_root")")"
    expected="module github.com/viralforge/mesh/services/$cluster_name/$dir_name"
    if [[ "$first_line" != "$expected" ]]; then
      errors+=("Invalid go.mod module path for $svc: got '$first_line', expected '$expected'")
    fi
  fi

  k8s_files=(
    "$service_root/deploy/k8s/deployment.yaml"
    "$service_root/deploy/k8s/service.yaml"
    "$service_root/deploy/k8s/configmap.yaml"
    "$service_root/deploy/k8s/hpa.yaml"
    "$service_root/deploy/k8s/pdb.yaml"
  )
  kf=""
  for kf in "${k8s_files[@]}"; do
    [[ -f "$kf" ]] || continue

    name_val="$(metadata_name "$kf")"
    if [[ -z "$name_val" ]]; then
      errors+=("Missing metadata.name in manifest: ${kf#$ROOT_PATH/}")
    elif ! is_rfc1123_label "$name_val"; then
      errors+=("Invalid Kubernetes metadata.name '${name_val}' in ${kf#$ROOT_PATH/}; expected lowercase RFC1123 label")
    fi

    while IFS= read -r app_value; do
      [[ -z "$app_value" ]] && continue
      if ! is_rfc1123_label "$app_value"; then
        errors+=("Invalid Kubernetes app label/selector '${app_value}' in ${kf#$ROOT_PATH/}; expected lowercase RFC1123 label")
      fi
    done < <(grep -E '^[[:space:]]*app:[[:space:]]*' "$kf" | sed -E 's/^[[:space:]]*app:[[:space:]]*//')
  done
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
