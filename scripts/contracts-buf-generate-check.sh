#!/usr/bin/env bash
set -euo pipefail

ROOT_PATH="mesh"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --root-path) ROOT_PATH="$2"; shift 2 ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

ROOT_ABS="$(cd "$ROOT_PATH" && pwd -P)"
if ! GIT_TOP="$(git -C "$ROOT_ABS" rev-parse --show-toplevel 2>/dev/null)"; then
  echo "contracts-buf-generate-check: failed to resolve git top-level for ROOT_PATH=$ROOT_PATH" >&2
  exit 1
fi
GIT_TOP="$(cd "$GIT_TOP" && pwd -P)"
if [[ "$ROOT_ABS" != "$GIT_TOP" ]]; then
  echo "contracts-buf-generate-check: must run with --root-path set to mesh git root (expected=$GIT_TOP actual=$ROOT_ABS)" >&2
  exit 1
fi

bash "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/contracts-buf-generate.sh" --root-path "$ROOT_PATH"

if ! git diff --exit-code -- "$ROOT_PATH/contracts/gen/go"; then
  echo "contracts-buf-generate-check: generated artifacts are out of date" >&2
  exit 1
fi

echo "contracts-buf-generate-check: passed"
