#!/usr/bin/env bash
set -euo pipefail
case "${CARGO_TARGET_DIR:-}" in
  ""|/*) ;;
  *) export CARGO_TARGET_DIR="$PWD/$CARGO_TARGET_DIR" ;;
esac
case "${CARGO_HOME:-}" in
  ""|/*) ;;
  *) export CARGO_HOME="$PWD/$CARGO_HOME" ;;
esac
cd "$(dirname "$0")/../.."
workspace=$(pwd -P)
target=${CARGO_TARGET_DIR:-$PWD/target}
mkdir -p "$target"
export CARGO_TARGET_DIR=$(cd "$target" && pwd -P)
(cd umpire && bash scripts/tests/build-guests.sh)
bash "$workspace/scripts/guest.sh" umpire-temporal-guest "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_temporal_guest.wasm"
