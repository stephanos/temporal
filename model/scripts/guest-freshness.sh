#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -ge 2 ]; then
  case "$2" in
    /*) ;;
    *) set -- "$1" "$PWD/$2" "${@:3}" ;;
  esac
fi
case "${CARGO_TARGET_DIR:-}" in
  ""|/*) ;;
  *) export CARGO_TARGET_DIR="$PWD/$CARGO_TARGET_DIR" ;;
esac
model=$(cd "$(dirname "$0")/.." && pwd -P)
case "${1:-}" in
  umpire-stock-guest|umpire-kettle-guest|umpire-control-guest|umpire-caught-panic-guest|umpire-overflow-guest) cd "$model/umpire" ;;
  *) cd "$model" ;;
esac
exec /bin/bash "$model/umpire/scripts/guest-freshness.sh" "$@"
