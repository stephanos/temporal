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
cd "$(dirname "$0")/.."
helper=$(/bin/bash "$PWD/scripts/xtask.sh")
exec "$helper" guest "$crate" "$output" "${@:3}"
