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
cd "$(dirname "$0")/.."
exec /bin/bash "$PWD/umpire/scripts/guest-freshness.sh" "$@"
