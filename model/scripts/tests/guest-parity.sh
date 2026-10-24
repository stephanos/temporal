#!/usr/bin/env bash
set -euo pipefail
: "${UMPIRE_WASM:?set the stock guest built with profile.guest}"
: "${UMPIRE_STATEFUL_WASM:?set the control guest built with profile.guest}"
: "${UMPIRE_CAUGHT_PANIC_WASM:?set the caught-panic guest built with profile.guest}"
: "${UMPIRE_OVERFLOW_WASM:?set the overflow guest built with profile.guest}"
: "${UMPIRE_PANIC_NATIVE:?set the native panic-control-reports output directory}"
cd "$(dirname "$0")/../.."
workspace=$(pwd -P)
CGO_ENABLED=0 mise exec -- go -C "$workspace/umpire/hosts/go" test -tags test_dep \
  -count=1 -v ./...
CGO_ENABLED=0 mise exec -- go -C "$workspace/umpire/reference/kettle" test -tags test_dep \
  -count=1 -v ./...
