#!/usr/bin/env bash
set -euo pipefail

ROOT_PATH="mesh"
# Script executes from "$ROOT_PATH/contracts", so point to repo-level git directory.
AGAINST="../.git#branch=main,subdir=contracts/proto"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --root-path) ROOT_PATH="$2"; shift 2 ;;
    --against) AGAINST="$2"; shift 2 ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

pushd "$ROOT_PATH/contracts" >/dev/null
buf breaking --against "$AGAINST"
popd >/dev/null

echo "contracts-buf-breaking: passed"
