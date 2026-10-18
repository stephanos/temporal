#!/usr/bin/env bash
# Times each step of the model2 gate (fn-210 R30) and warns when the feedback loop exceeds its budget.
# Uses bash's `time` so it works on macOS and Linux alike. It warns rather than fails: timing is
# machine-dependent, and a breach is fixed or excepted by the owner, not retried until green.
set -euo pipefail
cd "$(dirname "$0")/.."

INNER_BUDGET=${UMPIRE_INNER_BUDGET:-3}   # seconds: edit one crate, rebuild, run its fixtures
GATE_BUDGET=${UMPIRE_GATE_BUDGET:-30}    # seconds: the whole local gate, warm

TIMEFORMAT=%R
step() {
  local name=$1; shift
  local secs
  secs=$({ time "$@" >/dev/null 2>&1; } 2>&1 | tail -n1)
  printf '%-32s %6ss\n' "$name" "$secs"
  echo "$secs"
}

inner_file=umpire/checker/src/lib.rs
total=0
printf '%-32s %7s\n' "step (warm)" "time"
out=$(step "fmt --check" mise exec -- cargo fmt --all --check); echo "${out%$'\n'*}"; total=$(awk -v a="$total" -v b="${out##*$'\n'}" 'BEGIN{print a+b}')
out=$(step "clippy -D warnings" mise exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings); echo "${out%$'\n'*}"; total=$(awk -v a="$total" -v b="${out##*$'\n'}" 'BEGIN{print a+b}')
out=$(step "doc (warnings denied)" env RUSTDOCFLAGS="-D warnings" mise exec -- cargo doc --workspace --no-deps --quiet); echo "${out%$'\n'*}"; total=$(awk -v a="$total" -v b="${out##*$'\n'}" 'BEGIN{print a+b}')
out=$(step "test --workspace" mise exec -- cargo test --workspace --quiet); echo "${out%$'\n'*}"; total=$(awk -v a="$total" -v b="${out##*$'\n'}" 'BEGIN{print a+b}')
out=$(step "wasm32-wasip2 build" mise exec -- cargo build --target wasm32-wasip2 -p umpire-dsl -p umpire-checker --quiet); echo "${out%$'\n'*}"; total=$(awk -v a="$total" -v b="${out##*$'\n'}" 'BEGIN{print a+b}')

touch "$inner_file"
out=$(step "inner loop (checker edit+test)" mise exec -- cargo test -p umpire-checker --quiet); echo "${out%$'\n'*}"; inner=${out##*$'\n'}

printf '%-32s %6ss  (budget %ss)\n' "gate total" "$total" "$GATE_BUDGET"
printf '%-32s %6ss  (budget %ss)\n' "inner loop" "$inner" "$INNER_BUDGET"
awk -v t="$total" -v g="$GATE_BUDGET" 'BEGIN{ if (t > g) print "warning: gate over budget (fn-210 R30)" }'
awk -v i="$inner" -v b="$INNER_BUDGET" 'BEGIN{ if (i > b) print "warning: inner loop over budget (fn-210 R30)" }'
