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

bash "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/contracts-buf-generate.sh" --root-path "$ROOT_PATH"

if ! git diff --exit-code -- "$ROOT_PATH/contracts/gen/go"; then
  echo "contracts-buf-generate-check: generated artifacts are out of date" >&2
  exit 1
fi

echo "contracts-buf-generate-check: passed"
