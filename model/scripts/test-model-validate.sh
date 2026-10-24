#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."

# Controls use the shipping Make recipes and timer, with disposable child work only.
control=$(mktemp -d)
trap 'rm -rf "${control:?}"' EXIT
mkdir -p "$control/model/scripts" "$control/bin" "$control/output"
cp model/scripts/time-gate.sh "$control/model/scripts/"
printf 'control source\n' > "$control/model/inner.rs"
export UMPIRE_INNER_SOURCE=inner.rs UMPIRE_INNER_CRATE=control-checker
export CONTROL_LOG="$control/calls" TMPDIR="$control/output"
export UMPIRE_GATE_BUDGET=-1 UMPIRE_INNER_BUDGET=-1
unset UMPIRE_BACKENDS UMPIRE_APALACHE MAKEFLAGS MFLAGS MAKEOVERRIDES
printf 'COLOR = "%%s\\n"\n' > "$control/Makefile"
awk '/^model-(time|validate):/ {copy=1} copy {print} copy && /^$/ {copy=0}' Makefile >> "$control/Makefile"
cat >> "$control/Makefile" <<'MAKE'
.PHONY: model-check
model-check:
	@printf 'gate:%s:%s\n' "$${UMPIRE_BACKENDS-unset}" "$${UMPIRE_APALACHE-unset}" >> "$$CONTROL_LOG"
	@printf 'gate stdout diagnostic\n'; printf 'gate stderr diagnostic\n' >&2; exit "$${CONTROL_GATE_STATUS:-0}"
MAKE
cat > "$control/bin/mise" <<'CHILD'
#!/usr/bin/env bash
set -euo pipefail
[ "$*" = 'exec -- cargo test -p control-checker --quiet' ]
printf 'benchmark:%s:%s\n' "${UMPIRE_BACKENDS-unset}" "${UMPIRE_APALACHE-unset}" >> "$CONTROL_LOG"
printf 'benchmark stdout diagnostic\n'
printf 'benchmark stderr diagnostic\n' >&2
exit "${CONTROL_BENCHMARK_STATUS:-0}"
CHILD
chmod +x "$control/bin/mise"
export PATH="$control/bin:$PATH"
make_bin=$(command -v make)
cd "$control"

run_control() {
  local expected_status=$1 expected_calls=$2; shift 2
  local status=0 output retained actual
  : > "$CONTROL_LOG"
  output=$("$@" 2>&1) || status=$?
  if [ "$status" -ne "$expected_status" ]; then
    printf 'orchestration control expected exit %s, got %s: %s\n' "$expected_status" "$status" "$output" >&2
    exit 1
  fi
  actual=$(cat "$CONTROL_LOG")
  if [ "$actual" != "$expected_calls" ]; then
    printf 'orchestration control expected calls <%s>, got <%s>\n%s\n' "$expected_calls" "$actual" "$output" >&2
    exit 1
  fi
  for retained in "$TMPDIR"/*; do
    if [ -e "$retained" ]; then
      printf 'wrapper retained temporary output: %s\n' "$retained" >&2
      exit 1
    fi
  done
  if [ "$expected_status" -eq 0 ]; then
    for text in 'gate over soft target' 'inner loop over soft target'; do
      case "$output" in *"$text"*) ;; *) printf 'missing soft warning: %s\n' "$output" >&2; exit 1 ;; esac
    done
  else
    case "$expected_calls" in *benchmark*) stage=benchmark ;; *) stage=gate ;; esac
    for stream in stdout stderr; do
      case "$output" in *"$stage $stream diagnostic"*) ;; *) printf 'lost child diagnostics: %s\n' "$output" >&2; exit 1 ;; esac
    done
    case "$output" in *'over soft target'*) printf 'failed wrapper reported success budgets: %s\n' "$output" >&2; exit 1 ;; esac
  fi
}

for conflict in absent environment command-line makeflags command-line-makeflags; do
  for apalache in skip require; do
    calls="gate:require:$apalache"$'\n'"benchmark:require:$apalache"
    case "$conflict" in
      absent) run_control 0 "$calls" "$make_bin" UMPIRE_APALACHE="$apalache" model-validate ;;
      environment) run_control 0 "$calls" env UMPIRE_BACKENDS=skip UMPIRE_APALACHE="$apalache" "$make_bin" model-validate ;;
      command-line) run_control 0 "$calls" "$make_bin" UMPIRE_BACKENDS=skip UMPIRE_APALACHE="$apalache" model-validate ;;
      makeflags) run_control 0 "$calls" env MAKEFLAGS="UMPIRE_BACKENDS=skip UMPIRE_APALACHE=$apalache" "$make_bin" model-validate ;;
      command-line-makeflags) run_control 0 "$calls" "$make_bin" "MAKEFLAGS=UMPIRE_BACKENDS=skip UMPIRE_APALACHE=$apalache" model-validate ;;
    esac
  done
done
for mode in unset skip require; do
  calls="gate:$mode:unset"$'\n'"benchmark:$mode:unset"
  if [ "$mode" = unset ]; then
    run_control 0 "$calls" "$make_bin" model-time
  else
    run_control 0 "$calls" "$make_bin" UMPIRE_BACKENDS="$mode" model-time
  fi
done
run_control 2 'gate:require:unset' env CONTROL_GATE_STATUS=37 "$make_bin" model-validate
run_control 2 $'gate:require:unset\nbenchmark:require:unset' env CONTROL_BENCHMARK_STATUS=37 "$make_bin" model-validate
run_control 37 $'gate:unset:unset\nbenchmark:unset:unset' env CONTROL_BENCHMARK_STATUS=37 bash model/scripts/time-gate.sh
printf 'Model orchestration controls passed (fake children; no execution evidence).\n'
