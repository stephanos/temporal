#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."

# Execute shipping recipe boundaries without starting Rust or Go builds.
control=$(mktemp -d)
trap 'rm -rf "${control:?}"' EXIT
mkdir -p "$control/model/scripts" "$control/model/umpire/scripts/tests" "$control/model/scripts/tests" "$control/bin"
cp model/scripts/time-gate.sh "$control/model/scripts/"
cp model/scripts/tests/build-guests.sh "$control/model/scripts/tests/"
cat > "$control/model/umpire/scripts/tests/build-guests.sh" <<'CHILD'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$CARGO_TARGET_DIR" > "$CONTROL_BUILDER_TARGET"
CHILD
printf '#!/usr/bin/env bash\nset -euo pipefail\n' > "$control/model/scripts/guest.sh"
awk '
  /^RUST_MODEL_(STEP|WASM_ENV) =/ { print }
  /call RUST_MODEL_STEP,((fmt|clippy|docs|tests)(-framework)?|wasm32-wasip1|core-wasm-clippy)\)/ {
    stage=$0
    sub(/^.*call RUST_MODEL_STEP,/, "", stage)
    sub(/\).*$/, "", stage)
    print "control-" stage ":"
    print
  }
  /call RUST_MODEL_STEP,go-host\)/ { print "control-host:"; print }
  /call RUST_MODEL_STEP,kettle-go-host\)/ { print "control-kettle:"; print }
' Makefile > "$control/Makefile"
cat > "$control/bin/mise" <<'CHILD'
#!/usr/bin/env bash
set -euo pipefail
printf '%s|%s|%s\n' "$PWD" "$*" "${UMPIRE_CURRENT_CONTRACT_NATIVE-unset}" >> "$CONTROL_LOG"
CHILD
chmod +x "$control/bin/mise"
export PATH="$control/bin:$PATH" CONTROL_LOG="$control/calls" CONTROL_BUILDER_TARGET="$control/builder-target"
unset MAKEFLAGS MFLAGS MAKEOVERRIDES CARGO_TARGET_DIR UMPIRE_CURRENT_CONTRACT_NATIVE
cd "$control"
control=$(pwd -P)
failures=0
expect_call() {
  local expected=$1; shift
  local actual
  : > "$CONTROL_LOG"
  "$@" > "$control/output" 2>&1
  actual=$(cat "$CONTROL_LOG")
  if [ "$actual" != "$expected" ]; then
    printf 'recipe wiring (%s): expected <%s>, got <%s>\n' "$*" "$expected" "$actual" >&2
    failures=$((failures + 1))
  fi
}

for timed in 0 1; do
  for stage in fmt clippy docs tests; do
    case "$stage" in
      fmt) command='exec -- cargo fmt --all --check' ;;
      clippy) command='exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings' ;;
      docs) command='exec -- cargo doc --workspace --no-deps --quiet' ;;
      tests) command='exec -- cargo test --workspace --locked --quiet' ;;
    esac
    expect_call "$control/model|$command|unset" make UMPIRE_TIME_STEPS="$timed" "control-$stage"
    expect_call "$control/model/umpire|$command|unset" make UMPIRE_TIME_STEPS="$timed" "control-$stage-framework"
  done
  expect_call "$control/model/umpire|exec -- cargo build --target wasm32-wasip1 -p umpire-dsl -p umpire-checker --quiet|unset" make UMPIRE_TIME_STEPS="$timed" control-wasm32-wasip1
  expect_call "$control/model/umpire|exec -- cargo clippy --target wasm32-wasip1 -p umpire-dsl -p umpire-checker --lib --locked -- -D warnings|unset" make UMPIRE_TIME_STEPS="$timed" control-core-wasm-clippy
  for target_mode in unset relative absolute; do
    case "$target_mode" in
      unset) unset CARGO_TARGET_DIR; native="$control/model/target/current-contract-native" ;;
      relative) export CARGO_TARGET_DIR=relative-target; native="$control/relative-target/current-contract-native" ;;
      absolute) export CARGO_TARGET_DIR="$control/absolute-target"; native="$control/absolute-target/current-contract-native" ;;
    esac
    bash model/scripts/tests/build-guests.sh
    builder_target=$(cat "$CONTROL_BUILDER_TARGET")
    if [ "$builder_target/current-contract-native" != "$native" ]; then
      printf 'builder target (%s): expected <%s>, got <%s>\n' "$target_mode" "${native%/current-contract-native}" "$builder_target" >&2
      exit 1
    fi
    expect_call "$control/model|exec -- go -C $control/model/umpire/hosts/go test -tags test_dep -count=1 ./...|$native" make UMPIRE_TIME_STEPS="$timed" control-host
    expect_call "$control/model|exec -- go -C $control/model/umpire/reference/kettle test -tags test_dep -count=1 ./...|$native" make UMPIRE_TIME_STEPS="$timed" control-kettle
  done
  unset CARGO_TARGET_DIR
done
[ "$failures" -eq 0 ]
printf 'Model recipe wiring controls passed (fake children; no execution evidence).\n'
