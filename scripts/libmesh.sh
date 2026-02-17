#!/usr/bin/env bash
set -euo pipefail

trim() {
  local s="${1-}"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

slugify() {
  local s="${1-}"
  s="${s,,}"
  s="$(printf '%s' "$s" | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
  printf '%s' "$s"
}

resolve_cluster_slug() {
  local title="${1-}"
  case "${title,,}" in
    "platform edge and reliability") printf 'platform-ops' ;;
    "integrations and external boundaries") printf 'integrations' ;;
    "trust, compliance, and risk") printf 'trust-compliance' ;;
    "data, ai, and decisioning") printf 'data-ai' ;;
    "financial transaction rails") printf 'financial-rails' ;;
    *) slugify "$title" ;;
  esac
}

fallback_cluster() {
  local category="${1-}"
  case "$category" in
    "Core Platform & Foundation") printf 'core-platform' ;;
    "Operational & Infrastructure") printf 'platform-ops' ;;
    "Financials & Economy") printf 'financial-rails' ;;
    "AI & Automation") printf 'data-ai' ;;
    "Analytics & Reporting") printf 'data-ai' ;;
    "Compliance & Data Governance") printf 'trust-compliance' ;;
    "Moderation & Compliance") printf 'trust-compliance' ;;
    "Developer Ecosystem & Integrations") printf 'integrations' ;;
    "Distribution & Tracking") printf 'integrations' ;;
    "Notifications & Alerts") printf 'integrations' ;;
    "Customer Success & Support") printf 'integrations' ;;
    "Community & Engagement") printf 'integrations' ;;
    "Internal Admin & Operations") printf 'core-platform' ;;
    *) printf 'integrations' ;;
  esac
}

append_map_line() {
  local map_name="$1"
  local key="$2"
  local value="$3"
  declare -n map_ref="$map_name"
  local current="${map_ref[$key]-}"
  if [[ -z "$current" ]]; then
    map_ref["$key"]="$value"
  else
    map_ref["$key"]+=$'\n'"$value"
  fi
}

sorted_unique_block() {
  local block="${1-}"
  if [[ -z "$block" ]]; then
    return 0
  fi
  printf '%s\n' "$block" | sed '/^[[:space:]]*$/d' | LC_ALL=C sort -u
}

map_sorted_unique() {
  local map_name="$1"
  local key="$2"
  declare -n map_ref="$map_name"
  local block="${map_ref[$key]-}"
  sorted_unique_block "$block"
}

csv_from_block() {
  local block="${1-}"
  local out=""
  local line=""
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    if [[ -z "$out" ]]; then
      out="$line"
    else
      out+=", $line"
    fi
  done <<< "$block"
  printf '%s' "$out"
}

bullet_or_none() {
  local block="${1-}"
  if [[ -z "$block" ]]; then
    printf '%s' "- none"
    return 0
  fi
  local out=""
  local line=""
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    out+="- $line"$'\n'
  done <<< "$block"
  printf '%s' "${out%$'\n'}"
}

write_file() {
  local path="$1"
  local content="$2"
  mkdir -p "$(dirname "$path")"
  printf '%s' "$content" > "$path"
}

emit_yaml_list() {
  local indent="$1"
  local block="$2"
  if [[ -z "$block" ]]; then
    printf '%s[]\n' "$indent"
    return 0
  fi
  local line=""
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    printf '%s- %s\n' "$indent" "$line"
  done <<< "$block"
}

directory_name() {
  local service_id="$1"
  local service_name="$2"
  printf '%s-%s' "$service_id" "$(slugify "$service_name")"
}

get_spec_summary() {
  local specs_root="$1"
  local service_id="$2"
  local spec_file=""
  spec_file="$(find "$specs_root" -maxdepth 1 -type f -name "${service_id}-*.md" | LC_ALL=C sort | head -n1 || true)"
  if [[ -z "$spec_file" ]]; then
    printf '%s' "See canonical service specification."
    return 0
  fi
  local line=""
  line="$(grep -m1 -E '^\*\*Description:\*\*' "$spec_file" || true)"
  if [[ -z "$line" ]]; then
    printf '%s' "See canonical service specification."
    return 0
  fi
  printf '%s' "$(printf '%s' "$line" | sed -E 's/^\*\*Description:\*\*[[:space:]]*//')"
}

load_microservices() {
  local path="$1"
  declare -g -a SERVICES=()
  declare -g -A SERVICE_ID=()
  declare -g -A SERVICE_NAME=()

  local rows=()
  local line=""
  local current=""
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^[[:space:]]{2}(M[0-9]{2}-[^:]+):[[:space:]]*$ ]]; then
      current="${BASH_REMATCH[1]}"
      continue
    fi
    if [[ "$line" =~ ^[[:space:]]{4}architecture:[[:space:]]+microservice[[:space:]]*$ ]] && [[ -n "$current" ]]; then
      local id="${current:0:3}"
      local name="${current#M[0-9][0-9]-}"
      local id_num=$((10#${id#M}))
      rows+=("$(printf '%03d|%s|%s|%s' "$id_num" "$current" "$id" "$name")")
    fi
  done < "$path"

  local sorted_rows=()
  if ((${#rows[@]} > 0)); then
    mapfile -t sorted_rows < <(printf '%s\n' "${rows[@]}" | LC_ALL=C sort)
  fi

  local row=""
  for row in "${sorted_rows[@]}"; do
    [[ -z "$row" ]] && continue
    local rest="${row#*|}"
    local key="${rest%%|*}"
    rest="${rest#*|}"
    local id="${rest%%|*}"
    local name="${rest#*|}"
    SERVICES+=("$key")
    SERVICE_ID["$key"]="$id"
    SERVICE_NAME["$key"]="$name"
  done
}

load_dependencies() {
  local path="$1"
  declare -g -A DEP_HTTP=()
  declare -g -A DEP_DBR=()
  declare -g -A DEP_EVENT_DEPS=()
  declare -g -A DEP_EVENT_PROVIDES=()

  local line=""
  local current=""
  local mode=""
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^(M[0-9]{2}-[^:]+):[[:space:]]*$ ]]; then
      current="${BASH_REMATCH[1]}"
      mode=""
      DEP_HTTP["$current"]="${DEP_HTTP[$current]-0}"
      continue
    fi
    [[ -z "$current" ]] && continue

    if [[ "$line" =~ ^[[:space:]]{2}provides:[[:space:]]*$ ]]; then
      mode="provides"
      continue
    fi
    if [[ "$line" =~ ^[[:space:]]{2}depends_on:[[:space:]]*$ ]]; then
      mode="depends"
      continue
    fi
    if [[ "$line" =~ ^[[:space:]]{2}depends_on:[[:space:]]*\[\][[:space:]]*$ ]]; then
      mode=""
      continue
    fi
    if [[ "$line" =~ ^[[:space:]]{4}-[[:space:]]+(.+)$ ]]; then
      local value
      value="$(trim "${BASH_REMATCH[1]}")"
      if [[ "$mode" == "provides" ]]; then
        if [[ "$value" == "http" ]]; then
          DEP_HTTP["$current"]="1"
        elif [[ "$value" == EVENT:* ]]; then
          append_map_line DEP_EVENT_PROVIDES "$current" "${value#EVENT:}"
        fi
      elif [[ "$mode" == "depends" ]]; then
        if [[ "$value" == DBR:* ]]; then
          append_map_line DEP_DBR "$current" "${value#DBR:}"
        elif [[ "$value" == EVENT:* ]]; then
          append_map_line DEP_EVENT_DEPS "$current" "${value#EVENT:}"
        fi
      fi
    fi
  done < "$path"
}

load_categories_from_profile() {
  local path="$1"
  declare -g -A CATEGORY_MAP=()
  local line=""
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^\|[[:space:]]*(M[0-9]{2}-[^|]+)[[:space:]]*\|[[:space:]]*([^|]+)[[:space:]]*\|[[:space:]]*\*\*(Microservice|Monolith)\*\*[[:space:]]*\| ]]; then
      local svc
      local category
      local runtime
      svc="$(trim "${BASH_REMATCH[1]}")"
      category="$(trim "${BASH_REMATCH[2]}")"
      runtime="$(trim "${BASH_REMATCH[3]}")"
      if [[ "$runtime" == "Microservice" ]]; then
        CATEGORY_MAP["$svc"]="$category"
      fi
    fi
  done < "$path"
}

load_suggested_clusters_from_profile() {
  local path="$1"
  declare -g -A SUGGESTED_CLUSTER_MAP=()
  local in_section="0"
  local current_cluster=""
  local line=""
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" =~ ^##[[:space:]]+Suggested[[:space:]]+Microservice[[:space:]]+Clusters[[:space:]]*$ ]]; then
      in_section="1"
      continue
    fi
    if [[ "$in_section" == "1" ]] && [[ "$line" =~ ^##[[:space:]]+ ]]; then
      break
    fi
    [[ "$in_section" == "0" ]] && continue

    if [[ "$line" =~ ^###[[:space:]]+(.+)[[:space:]]*$ ]]; then
      current_cluster="$(resolve_cluster_slug "$(trim "${BASH_REMATCH[1]}")")"
      continue
    fi
    if [[ "$line" =~ ^-[[:space:]]+(M[0-9]{2}-[A-Za-z0-9-]+)[[:space:]]*$ ]]; then
      local svc="${BASH_REMATCH[1]}"
      if [[ -n "$current_cluster" && -z "${SUGGESTED_CLUSTER_MAP[$svc]-}" ]]; then
        SUGGESTED_CLUSTER_MAP["$svc"]="$current_cluster"
      fi
    fi
  done < "$path"
}

build_clustered_maps() {
  declare -g -A SVC_CLUSTER=()
  declare -g -A SVC_CATEGORY=()
  declare -g -A SVC_DIR=()
  local svc=""
  for svc in "${SERVICES[@]}"; do
    local category="${CATEGORY_MAP[$svc]-Uncategorized}"
    local cluster="${SUGGESTED_CLUSTER_MAP[$svc]-}"
    if [[ -z "$cluster" ]]; then
      cluster="$(fallback_cluster "$category")"
    fi
    SVC_CATEGORY["$svc"]="$category"
    SVC_CLUSTER["$svc"]="$cluster"
    SVC_DIR["$svc"]="$(directory_name "${SERVICE_ID[$svc]}" "${SERVICE_NAME[$svc]}")"
  done
}
