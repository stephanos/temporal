#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
control=$(mktemp -d)
trap 'rm -rf "${control:?}"' EXIT
mkdir -p "$control/model/umpire/scripts" "$control/model/dev/xtask/src" "$control/bin"
printf 'API_BINPB = control-api\n' > "$control/Makefile"
awk '/^##### model: Umpire in Rust/ { copy=1 } copy { print }' Makefile >> "$control/Makefile"
cat >> "$control/Makefile" <<'MAKE'
.PHONY: control-api
control-api:
	@test "$$UMPIRE_QUICK_CACHE:$$UMPIRE_CACHE_DISABLE" = 0:1
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
exit "${CONTROL_TEMPORAL_STATUS:-0}"
CHILD
chmod +x "$control/bin/mise" "$control/bin/framework"
export PATH="$control/bin:$PATH" CONTROL_LOG="$control/calls" CONTROL_FRAMEWORK="$control/bin/framework" CONTROL_ROOT="$control"
export CARGO_HOME=relative-home CARGO_TARGET_DIR=relative-target
unset MAKEFLAGS MFLAGS MAKEOVERRIDES
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
