#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$ROOT_DIR/scripts"
SPEC_ROOT="$ROOT_DIR/../viralForge/specs"
if [[ ! -d "$SPEC_ROOT" ]]; then
  SPEC_ROOT="$ROOT_DIR/viralForge/specs"
fi

echo "[gate-mesh1] validating mesh scaffold structure..."
bash "$SCRIPTS_DIR/validate-mesh-structure.sh" \
  --root-path "$ROOT_DIR" \
  --architecture-map-path "$SPEC_ROOT/service-architecture-map.yaml"

echo "[gate-mesh1] validating generated mesh index artifacts..."
bash "$SCRIPTS_DIR/generate-mesh-index.sh" \
  --root-path "$ROOT_DIR" \
  --architecture-map-path "$SPEC_ROOT/service-architecture-map.yaml" \
  --dependencies-path "$SPEC_ROOT/dependencies.yaml" \
  --deployment-profile-path "$SPEC_ROOT/service-deployment-profile.md" \
  --check

echo "[gate-mesh1] PASS"
