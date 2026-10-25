#!/usr/bin/env bash
set -euo pipefail
unset BASH_ENV
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
if [ "${1:-}" = historical ]; then
  exec python3 .flow/tmp/fn254-baseline/probe/historical.py
fi
exec python3 .flow/tmp/fn254-baseline/probe/measure.py "$@"
