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
target=$(cd "$target" && pwd -P)
export CARGO_TARGET_DIR="$target"
for pair in \
  umpire-stock-guest:umpire_stock_guest \
  umpire-kettle-guest:umpire_kettle_guest \
  umpire-control-guest:stateful_control \
  umpire-caught-panic-guest:caught_panic_control \
  umpire-overflow-guest:overflow_control; do
  bash "$workspace/scripts/guest.sh" "${pair%%:*}" "$target/wasm32-wasip1/guest/${pair#*:}.wasm"
done
mise exec -- cargo run --locked --profile guest -p umpire-caught-panic-guest \
  --bin panic-control-reports -- "$target/panic-native"
mise exec -- cargo run --locked --profile guest -p umpire-control-guest \
  --bin current-contract-reports -- "$target/current-contract-native"
