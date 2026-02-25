#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MESH_ROOT="$ROOT_DIR/.."
SPEC_ROOT="$MESH_ROOT/../viralForge/specs"
if [[ ! -d "$SPEC_ROOT" ]]; then
  SPEC_ROOT="$MESH_ROOT/viralForge/specs"
fi

echo "[mesh-gates] running gate-mesh1..."
bash "$ROOT_DIR/gate-mesh1.sh"
echo "[mesh-gates] running gate-mesh-events-contracts..."
bash "$ROOT_DIR/gate-mesh-events-contracts.sh" \
  --root-path "$MESH_ROOT" \
  --architecture-map-path "$SPEC_ROOT/service-architecture-map.yaml" \
  --dependencies-path "$SPEC_ROOT/dependencies.yaml"
echo "[mesh-gates] all mesh gates passed"
