#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Exercise the actual helper in the command-substitution context used by the gate.
definition=$(sed -n '/^step() {$/,/^}$/p' scripts/time-gate.sh)
for control in true false 'exit 37'; do
  expected=0
  case "$control" in
    false) expected=1 ;;
    'exit 37') expected=37 ;;
  esac
  rc=0
  /bin/bash -euo pipefail -c "$definition
TIMEFORMAT=%R
out=\$(step intentional_control /bin/bash -c \"\$1\")
printf '%s\\n' \"\$out\"
" -- "$control" >/dev/null 2>&1 || rc=$?
  if [ "$rc" -ne "$expected" ]; then
    printf 'timing control %s: expected exit %s, got %s\n' "$control" "$expected" "$rc" >&2
    exit 1
  fi
done
