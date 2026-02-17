#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$ROOT_DIR/scripts"

echo "[gate-mesh1] validating mesh scaffold structure..."
bash "$SCRIPTS_DIR/validate-mesh-structure.sh" --root-path "$ROOT_DIR"

echo "[gate-mesh1] validating generated mesh index artifacts..."
bash "$SCRIPTS_DIR/generate-mesh-index.sh" --root-path "$ROOT_DIR" --check

echo "[gate-mesh1] PASS"
