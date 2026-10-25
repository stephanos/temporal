#!/usr/bin/env bash
set -euo pipefail
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
if [ "$#" -lt 2 ] || [ "$#" -gt 3 ] || { [ "$#" -eq 3 ] && [ "$3" != --check ]; }; then
  printf 'usage: guest.sh <crate> <output> [--check]\n' >&2
  exit 1
fi
crate=$1
case "$2" in
  /*) output=$2 ;;
  *) output="$PWD/$2" ;;
esac
case "${CARGO_HOME:-}" in
  ""|/*) ;;
  *) export CARGO_HOME="$PWD/$CARGO_HOME" ;;
esac
if [ -n "${CARGO_HOME:-}" ] && [ -d "$CARGO_HOME" ]; then
  export CARGO_HOME=$(cd "$CARGO_HOME" && pwd -P)
fi
tool_root=$(cd "$(dirname "$0")/.." && pwd -P)
export UMPIRE_TOOL_ROOT="$tool_root"
helper=$(/bin/bash "$(dirname "$0")/xtask.sh")
exec "$helper" guest "$crate" "$output" "${@:3}"
