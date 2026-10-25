#!/usr/bin/env bash
set -euo pipefail
helper=${UMPIRE_XTASK:-}
if [ -z "$helper" ]; then
  helper=$(/bin/bash "$(dirname "$0")/xtask.sh")
fi
exec "$helper" retired-names "$@"
