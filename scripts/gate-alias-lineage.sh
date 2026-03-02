#!/usr/bin/env bash
set -euo pipefail

ROOT_PATH="mesh"
CATEGORY_MAP_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --root-path) ROOT_PATH="$2"; shift 2 ;;
    --category-map-path) CATEGORY_MAP_PATH="$2"; shift 2 ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$CATEGORY_MAP_PATH" ]]; then
  if [[ -f "$ROOT_PATH/../viralForge/specs/service-category-map.yaml" ]]; then
    CATEGORY_MAP_PATH="$ROOT_PATH/../viralForge/specs/service-category-map.yaml"
  else
    CATEGORY_MAP_PATH="$ROOT_PATH/viralForge/specs/service-category-map.yaml"
  fi
fi

if [[ ! -f "$CATEGORY_MAP_PATH" ]]; then
  echo "[gate-alias-lineage] category map missing: $CATEGORY_MAP_PATH" >&2
  exit 1
fi

declare -A expected_alias_to_successor=(
  ["M27"]="M08"
  ["M28"]="M07"
  ["M29"]="M09"
  ["M32"]="M11"
  ["M33"]="M12"
  ["M40"]="M13"
  ["M42"]="M14"
  ["M43"]="M15"
  ["M59"]="M03"
  ["M63"]="M50"
  ["M64"]="M51"
  ["M75"]="M65"
  ["M76"]="M16"
  ["M81"]="M18"
  ["M82"]="M19"
  ["M94"]="M30"
)

declare -A expected_supersedes=(
  ["M03"]="M59-Notification-Service"
  ["M07"]="M28-Editor-Dashboard-Service"
  ["M08"]="M27-Voting-Engine"
  ["M09"]="M29-Content-Marketplace-Service"
  ["M11"]="M32-Distribution-Tracking-Service"
  ["M12"]="M33-Fraud-Detection-Engine"
  ["M13"]="M40-Escrow-Service"
  ["M14"]="M42-Payout-Engine"
  ["M15"]="M43-Platform-Fee-Engine"
  ["M16"]="M76-API-Gateway"
  ["M18"]="M81-Cache-Service"
  ["M19"]="M82-Storage-Lifecycle-Management"
  ["M30"]="M94-Social-Integration-Service"
  ["M50"]="M63-Consent-Service"
  ["M51"]="M64-Data-Portability-Service"
  ["M65"]="M75-Churn-Prevention"
)

declare -A allow_set=()
declare -A supersedes_by_service=()
current_service=""
in_allowlist="0"

while IFS= read -r line; do
  if [[ "$line" =~ ^allow_missing_ids:[[:space:]]*$ ]]; then
    in_allowlist="1"
    continue
  fi
  if [[ "$line" =~ ^services:[[:space:]]*$ ]]; then
    in_allowlist="0"
    continue
  fi

  if [[ "$in_allowlist" == "1" && "$line" =~ ^[[:space:]]*-[[:space:]]*(M[0-9]{2})[[:space:]]*$ ]]; then
    allow_set["${BASH_REMATCH[1]}"]="1"
    continue
  fi

  if [[ "$line" =~ ^[[:space:]]{2}(M[0-9]{2}-[^:]+):[[:space:]]*$ ]]; then
    current_service="${BASH_REMATCH[1]}"
    continue
  fi

  if [[ -n "$current_service" && "$line" =~ ^[[:space:]]{4}supersedes:[[:space:]]*\[(.*)\][[:space:]]*$ ]]; then
    supersedes_by_service["$current_service"]="${BASH_REMATCH[1]}"
  fi
done < "$CATEGORY_MAP_PATH"

errors=()

for alias_id in "${!expected_alias_to_successor[@]}"; do
  if [[ -z "${allow_set[$alias_id]-}" ]]; then
    errors+=("allow_missing_ids missing deprecated alias ID: $alias_id")
  fi
done

for present_id in "${!allow_set[@]}"; do
  if [[ -n "${expected_alias_to_successor[$present_id]-}" ]]; then
    continue
  fi
  if [[ "$present_id" == "M90" || "$present_id" == "M93" ]]; then
    continue
  fi
  errors+=("allow_missing_ids contains unexpected ID: $present_id")
done

for alias_id in "${!expected_alias_to_successor[@]}"; do
  successor_id="${expected_alias_to_successor[$alias_id]}"
  if [[ -n "${allow_set[$successor_id]-}" ]]; then
    errors+=("allow_missing_ids incorrectly contains canonical successor ID: $successor_id")
  fi
done

for successor_id in "${!expected_supersedes[@]}"; do
  expected_deprecated="${expected_supersedes[$successor_id]}"
  successor_service=""
  for svc in "${!supersedes_by_service[@]}"; do
    if [[ "$svc" == "$successor_id"-* ]]; then
      successor_service="$svc"
      break
    fi
  done

  if [[ -z "$successor_service" ]]; then
    errors+=("Missing successor service entry for ID $successor_id in category map")
    continue
  fi

  supersedes_raw="${supersedes_by_service[$successor_service]-}"
  if [[ "$supersedes_raw" != *"$expected_deprecated"* ]]; then
    errors+=("$successor_service supersedes list missing expected deprecated lineage '$expected_deprecated'")
  fi
done

if ((${#errors[@]} > 0)); then
  echo "[gate-alias-lineage] FAILED"
  for err in "${errors[@]}"; do
    echo "$err" >&2
  done
  exit 1
fi

echo "[gate-alias-lineage] PASS (allowlist + supersedes lineage preserved)"
