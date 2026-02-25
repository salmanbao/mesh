#!/usr/bin/env bash
set -euo pipefail

BASE_SHA="${1:-}"
HEAD_SHA="${2:-HEAD}"

if [[ -z "$BASE_SHA" ]]; then
  BASE_SHA="$(git rev-parse "${HEAD_SHA}^" 2>/dev/null || true)"
fi

if [[ -z "$BASE_SHA" ]]; then
  echo "[]"
  exit 0
fi

mapfile -t changed_files < <(git diff --name-only "$BASE_SHA" "$HEAD_SHA")
if [[ "${#changed_files[@]}" -eq 0 ]]; then
  echo "[]"
  exit 0
fi

declare -A module_set=()

add_all_modules() {
  local dir=""
  for dir in platform contracts; do
    if [[ -f "$dir/go.mod" ]]; then
      module_set["$dir"]=1
    fi
  done
  while IFS= read -r dir; do
    [[ -z "$dir" ]] && continue
    module_set["$dir"]=1
  done < <(find services -mindepth 2 -maxdepth 2 -type d | LC_ALL=C sort)
}

for file in "${changed_files[@]}"; do
  if [[ "$file" == go.work || "$file" == go.work.sum ]]; then
    add_all_modules
    continue
  fi

  if [[ "$file" == platform/* ]]; then
    module_set["platform"]=1
    continue
  fi

  if [[ "$file" == contracts/* ]]; then
    module_set["contracts"]=1
    continue
  fi

  if [[ "$file" == services/*/*/* ]]; then
    module_root="$(printf '%s' "$file" | cut -d/ -f1-3)"
    if [[ -f "$module_root/go.mod" ]]; then
      module_set["$module_root"]=1
    fi
    continue
  fi
done

if [[ "${#module_set[@]}" -eq 0 ]]; then
  echo "[]"
  exit 0
fi

mapfile -t modules < <(printf '%s\n' "${!module_set[@]}" | LC_ALL=C sort)
printf '%s\n' "${modules[@]}" | awk '
  BEGIN { printf "["; first=1 }
  {
    gsub(/\\/,"\\\\",$0);
    gsub(/"/,"\\\"",$0);
    if (!first) printf ",";
    printf "\"" $0 "\"";
    first=0;
  }
  END { printf "]" }
'
