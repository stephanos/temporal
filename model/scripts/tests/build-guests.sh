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
if [ -n "${CARGO_HOME:-}" ] && [ -d "$CARGO_HOME" ]; then
  export CARGO_HOME=$(cd "$CARGO_HOME" && pwd -P)
fi
cd "$(dirname "$0")/../.."
workspace=$(pwd -P)
target=${CARGO_TARGET_DIR:-$PWD/target}
mkdir -p "$target"
export CARGO_TARGET_DIR=$(cd "$target" && pwd -P)
bash "$workspace/scripts/guest.sh" umpire-temporal-guest "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_temporal_guest.wasm"
