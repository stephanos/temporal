#!/usr/bin/env bash
# Times the model gate (fn-210 R30), recording overruns against soft targets.
# Small isolated overruns do not block completion; investigate substantial or repeated overruns.
set -euo pipefail
cd "$(dirname "$0")/.."

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

inner_crate=${UMPIRE_INNER_CRATE:-umpire-checker}
inner_file=${UMPIRE_INNER_SOURCE:-umpire/backends/native/src/lib.rs}
printf '%-32s %7s\n' "step (warm)" "time"
printf 'backend modes: UMPIRE_BACKENDS=%s UMPIRE_APALACHE=%s\n' "${UMPIRE_BACKENDS:-skip (default)}" "${UMPIRE_APALACHE:-skip (default)}"
printf 'gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check\n'
gate_log=$(mktemp)
trap 'rm -f "$gate_log"' EXIT
total=$({ time UMPIRE_TIME_STEPS=1 make -C .. model-check >"$gate_log" 2>&1; } 2>&1) || {
  rc=$?
  cat "$gate_log"
  exit "$rc"
}
cat "$gate_log"

test -f "$inner_file"
printf 'inner source: %s\ninner command: mise exec -- cargo test -p %s --quiet\n' "$inner_file" "$inner_crate"
touch "$inner_file"
out=$(step "inner loop (edit+test)" mise exec -- cargo test -p "$inner_crate" --quiet); echo "${out%$'\n'*}"; inner=${out##*$'\n'}

printf '%-32s %6ss  (soft target %ss)\n' "gate total" "$total" "$GATE_BUDGET"
printf '%-32s %6ss  (soft target %ss)\n' "inner loop" "$inner" "$INNER_BUDGET"
awk -v t="$total" -v g="$GATE_BUDGET" 'BEGIN{ if (t > g) print "note: gate over soft target (fn-210 R30)" }'
awk -v i="$inner" -v b="$INNER_BUDGET" 'BEGIN{ if (i > b) print "note: inner loop over soft target (fn-210 R30)" }'
