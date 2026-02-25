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

pushd "$ROOT_PATH/contracts" >/dev/null
buf lint
popd >/dev/null

echo "contracts-buf-lint: passed"
