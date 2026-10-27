#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
control=$(mktemp -d)
trap 'rm -rf "${control:?}"' EXIT
mkdir -p "$control/model/umpire/scripts" "$control/model/dev/xtask/src" "$control/bin"
printf 'API_BINPB = control-api\nPROTOC_GEN_GO = control-go\nPROTOC_GEN_GO_GRPC = control-grpc\n' > "$control/Makefile"
awk '/^##### model: Umpire in Rust/ { copy=1 } copy { print }' Makefile >> "$control/Makefile"
cat >> "$control/Makefile" <<'MAKE'
.PHONY: control-api control-go control-grpc
control-api control-go control-grpc:
	@test "$$UMPIRE_QUICK_CACHE:$$UMPIRE_CACHE_DISABLE" = 0:1
	@printf 'producer|%s\n' "$@" >> "$$CONTROL_LOG"
MAKE
cat > "$control/model/umpire/scripts/xtask.sh" <<'CHILD'
#!/usr/bin/env bash
printf '%s\n' "$CONTROL_FRAMEWORK"
CHILD
cat > "$control/bin/framework" <<'CHILD'
#!/usr/bin/env bash
set -euo pipefail
printf 'framework|%s|%s|%s|%s\n' "$PWD" "$*" "${UMPIRE_QUICK_CACHE-unset}" "${UMPIRE_CACHE_DISABLE-unset}" >> "$CONTROL_LOG"
[ "$PWD" = "$CONTROL_ROOT" ]
[ "$CARGO_HOME:$CARGO_TARGET_DIR" = relative-home:relative-target ]
exit "${CONTROL_FRAMEWORK_STATUS:-0}"
CHILD
cat > "$control/bin/mise" <<'CHILD'
#!/usr/bin/env bash
set -euo pipefail
printf 'temporal|%s|%s|%s|%s\n' "$PWD" "$*" "${UMPIRE_QUICK_CACHE-unset}" "${UMPIRE_CACHE_DISABLE-unset}" >> "$CONTROL_LOG"
[ "$PWD" = "$CONTROL_ROOT" ]
[ "$CARGO_HOME:$CARGO_TARGET_DIR" = relative-home:relative-target ]
if [[ "$*" = *' -- quick' ]] || [ "${UMPIRE_BACKENDS:-}" = require ] || [ "${UMPIRE_APALACHE:-}" = require ]; then
  [ "$CARGO_BUILD_JOBS" = 1 ] || { printf 'validation bootstrap expected one compiler, got %s\n' "$CARGO_BUILD_JOBS" >&2; exit 91; }
else
  [ "$CARGO_BUILD_JOBS" = 99 ]
fi
exit "${CONTROL_TEMPORAL_STATUS:-0}"
CHILD
chmod +x "$control/bin/mise" "$control/bin/framework"
export PATH="$control/bin:$PATH" CONTROL_LOG="$control/calls" CONTROL_FRAMEWORK="$control/bin/framework" CONTROL_ROOT="$control"
export CARGO_HOME=relative-home CARGO_TARGET_DIR=relative-target
export CARGO_BUILD_JOBS=99
unset MAKEFLAGS MFLAGS MAKEOVERRIDES UMPIRE_BACKENDS UMPIRE_APALACHE
cd "$control"
control=$(pwd -P)
for timed in 0 1; do
  : > "$CONTROL_LOG"
  UMPIRE_QUICK_CACHE=1 UMPIRE_CACHE_DISABLE=0 make UMPIRE_TIME_STEPS="$timed" UMPIRE_QUICK_CACHE=1 UMPIRE_CACHE_DISABLE=0 model-check > "$control/output" 2>&1
  expected=$(printf 'temporal|%s|exec -- cargo run --manifest-path %s/model/Cargo.toml --package temporal-xtask --bin temporal-xtask --profile dev -- gate|0|1' "$control" "$control")
  actual=$(cat "$CONTROL_LOG")
  if [ "$actual" != "$expected" ]; then
    printf 'combined gate wiring expected <%s>, got <%s>\n' "$expected" "$actual" >&2
    exit 1
  fi
done
for mode in required optional; do
  : > "$CONTROL_LOG"
  if [ "$mode" = required ]; then
    make UMPIRE_BACKENDS=require model-check > "$control/output" 2>&1
  else
    make UMPIRE_APALACHE=require model-check > "$control/output" 2>&1
  fi
  [ "$(wc -l < "$CONTROL_LOG" | tr -d ' ')" -eq 1 ]
done
: > "$CONTROL_LOG"
if ! make model-quick > "$control/output" 2>&1; then
  cat "$control/output" >&2
  exit 1
fi
[ "$(wc -l < "$CONTROL_LOG" | tr -d ' ')" -eq 1 ]
: > "$CONTROL_LOG"
make -j1 UMPIRE_QUICK_CACHE=1 UMPIRE_CACHE_DISABLE=0 model-prerequisites > "$control/output" 2>&1
[ "$(cat "$CONTROL_LOG")" = $'producer|control-go\nproducer|control-grpc\nproducer|control-api' ]
: > "$CONTROL_LOG"
status=0
CONTROL_TEMPORAL_STATUS=37 make model-check > "$control/output" 2>&1 || status=$?
[ "$status" -eq 2 ]
[ "$(wc -l < "$CONTROL_LOG" | tr -d ' ')" -eq 1 ]
for action in fmt update; do
  : > "$CONTROL_LOG"
  make "model-$action" > "$control/output" 2>&1
  [ "$(wc -l < "$CONTROL_LOG" | tr -d ' ')" -eq 2 ]
  head -1 "$CONTROL_LOG" | rg -q "framework\|.*\|$action\|"
  if [ "$action" = update ]; then
    expected=$(printf 'temporal|%s|exec -- cargo run --manifest-path %s/model/Cargo.toml --package temporal-xtask --bin temporal-xtask --profile dev -- test-run|unset|unset' "$control" "$control")
    [ "$(tail -1 "$CONTROL_LOG")" = "$expected" ]
  fi
  : > "$CONTROL_LOG"
  status=0
  CONTROL_FRAMEWORK_STATUS=37 make "model-$action" > "$control/output" 2>&1 || status=$?
  [ "$status" -eq 2 ]
  [ "$(wc -l < "$CONTROL_LOG" | tr -d ' ')" -eq 1 ]
done
printf 'Model recipe wiring controls passed (fake children; no execution evidence).\n'
