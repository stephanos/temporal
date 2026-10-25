#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
scratch=$(mktemp -d)
trap 'rm -rf "${scratch:?}"' EXIT
mkdir -p "$scratch/framework/scripts/tests" "$scratch/bin" "$scratch/caller/real cargo"
cp scripts/tests/build-guests.sh "$scratch/framework/scripts/tests/"
cat > "$scratch/framework/scripts/guest.sh" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[ "$PWD" = "$CONTROL_FRAMEWORK" ]
[ "$CARGO_HOME" = "$CONTROL_CALLER/real cargo" ]
printf 'guest|%s|%s\n' "$1" "$2" >> "$CONTROL_LOG"
[ "${FAIL_GUEST:-}" != "$1" ] || exit 37
TOOL
cat > "$scratch/bin/mise" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[ "$PWD" = "$CONTROL_FRAMEWORK" ]
[ "$CARGO_HOME" = "$CONTROL_CALLER/real cargo" ]
printf 'native|%s\n' "$*" >> "$CONTROL_LOG"
[ "${FAIL_NATIVE:-}" != "$9" ] || exit 42
TOOL
chmod +x "$scratch/bin/mise"
export PATH="$scratch/bin:$PATH" CONTROL_FRAMEWORK="$scratch/framework" CONTROL_CALLER="$scratch/caller" CONTROL_LOG="$scratch/calls"
ln -s 'real cargo' "$scratch/caller/cargo home"
cd "$scratch/caller"
export CARGO_HOME='cargo home' CARGO_TARGET_DIR='relative target'
bash "$scratch/framework/scripts/tests/build-guests.sh"
target="$scratch/caller/relative target"
cat > "$scratch/expected" <<EXPECTED
guest|umpire-stock-guest|$target/wasm32-wasip1/guest/umpire_stock_guest.wasm
guest|umpire-kettle-guest|$target/wasm32-wasip1/guest/umpire_kettle_guest.wasm
guest|umpire-control-guest|$target/wasm32-wasip1/guest/stateful_control.wasm
guest|umpire-caught-panic-guest|$target/wasm32-wasip1/guest/caught_panic_control.wasm
guest|umpire-overflow-guest|$target/wasm32-wasip1/guest/overflow_control.wasm
native|exec -- cargo run --locked --profile guest -p umpire-caught-panic-guest --bin panic-control-reports -- $target/panic-native
native|exec -- cargo run --locked --profile guest -p umpire-control-guest --bin current-contract-reports -- $target/current-contract-native
EXPECTED
cmp "$scratch/expected" "$scratch/calls"
: > "$scratch/calls"
status=0
FAIL_GUEST=umpire-control-guest bash "$scratch/framework/scripts/tests/build-guests.sh" || status=$?
[ "$status" -eq 37 ] && [ "$(wc -l < "$scratch/calls" | tr -d ' ')" -eq 3 ]
: > "$scratch/calls"
status=0
FAIL_NATIVE=umpire-caught-panic-guest bash "$scratch/framework/scripts/tests/build-guests.sh" || status=$?
[ "$status" -eq 42 ] && [ "$(wc -l < "$scratch/calls" | tr -d ' ')" -eq 6 ]
printf 'guest wiring controls: five guests, two native reports, caller paths and child failures passed (fake builders)\n'
