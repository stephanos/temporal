#!/usr/bin/env bash
set -euo pipefail
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
if [ "$#" -ne 2 ]; then
  printf 'usage: guest-freshness.sh <crate> <output>\n' >&2
  exit 1
fi
crate=$1
case "$2" in
  /*) output=$2 ;;
  *) output="$PWD/$2" ;;
esac
cd "$(dirname "$0")/.."
helper=$(/bin/bash "$PWD/scripts/xtask.sh")
exec "$helper" guest-freshness "$crate" "$output"
