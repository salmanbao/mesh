#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "[mesh-gates] running gate-mesh1..."
bash "$ROOT_DIR/gate-mesh1.sh"
echo "[mesh-gates] all mesh gates passed"
