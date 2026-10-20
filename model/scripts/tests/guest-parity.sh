#!/usr/bin/env bash
set -euo pipefail
: "${UMPIRE_WASM:?set the stock guest built with profile.guest}"
: "${UMPIRE_STATEFUL_WASM:?set the control guest built with profile.guest}"
cd "$(dirname "$0")/../.."
workspace=$(pwd -P)
overlay=$(mktemp)
trap 'rm -f "$overlay"' EXIT
python3 - "$workspace" "$overlay" <<'PY'
import json
from pathlib import Path
import sys

root = Path(sys.argv[1])
Path(sys.argv[2]).write_text(json.dumps({"Replace": {
    str(root / "umpire/hosts/go/guest_profile_test.go"):
    str(root / "scripts/tests/guest_profile_test.go.fixture"),
}}))
PY
CGO_ENABLED=0 mise exec -- go -C "$workspace/umpire/hosts/go" test -tags test_dep \
  -overlay "$overlay" -count=1 -v ./...
