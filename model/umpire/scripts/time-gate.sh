#!/usr/bin/env bash
# Times the model gate (fn-210 R30), recording overruns against soft targets.
# Small isolated overruns do not block completion; investigate substantial or repeated overruns.
set -euo pipefail

INNER_BUDGET=${UMPIRE_INNER_BUDGET:-10}  # seconds: edit one crate, rebuild, run its fixtures
GATE_BUDGET=${UMPIRE_GATE_BUDGET:-60}    # seconds: the whole local gate, warm

TIMEFORMAT=%R
step() {
  (
    local name=$1; shift
    local secs output status=0
    output=$(mktemp)
    trap 'rm -f "$output"' EXIT
    secs=$({ time "$@" >"$output" 2>&1; } 2>&1 | tail -n1) || status=$?
    if [ "$status" -ne 0 ]; then
      cat "$output" >&2
      exit "$status"
    fi
    printf '%-32s %6ss\n' "$name" "$secs"
    echo "$secs"
  )
}

if [ "${1:-}" = --step ]; then
  shift
  out=$(step "$@")
  printf '%s\n' "${out%$'\n'*}"
  exit 0
fi

printf 'usage: time-gate.sh --step <name> <command> [arguments...]\n' >&2
exit 1
