#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=mesh/scripts/libmesh.sh
source "$SCRIPT_DIR/libmesh.sh"

ROOT_PATH="mesh"
ARCHITECTURE_MAP_PATH="viralForge/specs/service-architecture-map.yaml"
DEPENDENCIES_PATH="viralForge/specs/dependencies.yaml"
IMPLEMENTED_SERVICES_PATH=""
CONTRACTS_EVENTS_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --root-path) ROOT_PATH="$2"; shift 2 ;;
    --architecture-map-path) ARCHITECTURE_MAP_PATH="$2"; shift 2 ;;
    --dependencies-path) DEPENDENCIES_PATH="$2"; shift 2 ;;
    --implemented-services-path) IMPLEMENTED_SERVICES_PATH="$2"; shift 2 ;;
    --contracts-events-path) CONTRACTS_EVENTS_PATH="$2"; shift 2 ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$IMPLEMENTED_SERVICES_PATH" ]]; then
  IMPLEMENTED_SERVICES_PATH="$ROOT_PATH/tooling/manifests/implemented-services.yaml"
fi
if [[ -z "$CONTRACTS_EVENTS_PATH" ]]; then
  CONTRACTS_EVENTS_PATH="$ROOT_PATH/contracts/events"
fi

if [[ ! -f "$IMPLEMENTED_SERVICES_PATH" ]]; then
  echo "Implemented services registry missing: $IMPLEMENTED_SERVICES_PATH" >&2
  exit 1
fi
if [[ ! -d "$CONTRACTS_EVENTS_PATH" ]]; then
  echo "Contracts events directory missing: $CONTRACTS_EVENTS_PATH" >&2
  exit 1
fi

load_microservices "$ARCHITECTURE_MAP_PATH"
load_dependencies "$DEPENDENCIES_PATH"

implemented=()
while IFS= read -r line; do
  if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*(M[0-9]{2}-[^[:space:]]+)[[:space:]]*$ ]]; then
    implemented+=("${BASH_REMATCH[1]}")
  fi
done < "$IMPLEMENTED_SERVICES_PATH"

if [[ "${#implemented[@]}" -eq 0 ]]; then
  echo "[gate-mesh-events-contracts] no implemented services listed; skipping strict event contract check"
  exit 0
fi

declare -A known_micro=()
svc=""
for svc in "${SERVICES[@]}"; do
  known_micro["$svc"]="1"
done

missing=()
for svc in "${implemented[@]}"; do
  if [[ -z "${known_micro[$svc]-}" ]]; then
    missing+=("Implemented service is not a known microservice: $svc")
    continue
  fi

  evt=""
  while IFS= read -r evt; do
    [[ -z "$evt" ]] && continue
    if [[ ! -f "$CONTRACTS_EVENTS_PATH/$evt.json" ]]; then
      missing+=("Missing event contract for dependency '$evt' required by $svc")
    fi
  done < <(map_sorted_unique DEP_EVENT_DEPS "$svc")

  while IFS= read -r evt; do
    [[ -z "$evt" ]] && continue
    if [[ ! -f "$CONTRACTS_EVENTS_PATH/$evt.json" ]]; then
      missing+=("Missing event contract for provided event '$evt' emitted by $svc")
    fi
  done < <(map_sorted_unique DEP_EVENT_PROVIDES "$svc")
done

if ((${#missing[@]} > 0)); then
  echo "[gate-mesh-events-contracts] FAILED"
  issue=""
  for issue in "${missing[@]}"; do
    echo "$issue" >&2
  done
  exit 1
fi

echo "[gate-mesh-events-contracts] PASS (${#implemented[@]} implemented services checked)"
