#!/usr/bin/env bash
set -euxo pipefail
root=$1
cd "$root/source-a"
export CARGO_BUILD_JOBS=2
export CARGO_TARGET_DIR="$root/target-a"
unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS
bash model/scripts/tests/build-guests.sh
cmp "$root/stock-a.wasm" "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_stock_guest.wasm"
cmp "$root/temporal-a.wasm" "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_temporal_guest.wasm"
export UMPIRE_WASM="$root/stock-a.wasm"
export UMPIRE_STATEFUL_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/stateful_control.wasm"
export UMPIRE_CAUGHT_PANIC_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/caught_panic_control.wasm"
export UMPIRE_OVERFLOW_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/overflow_control.wasm"
export UMPIRE_PANIC_NATIVE="$CARGO_TARGET_DIR/panic-native"
bash model/scripts/tests/guest-parity.sh
test ! -e "$root/temporal-host-tests"
cp -R model/temporal/hosts/go "$root/temporal-host-tests"
cp "$root/temporal-a.wasm" "$root/temporal-host-tests/umpire_temporal_guest.wasm"
mise exec -- go -C "$root/temporal-host-tests" mod edit \
  -replace "umpire.local/model=$root/source-a/model/umpire/hosts/go"
cmp "$root/temporal-a.wasm" "$root/temporal-host-tests/umpire_temporal_guest.wasm"
CGO_ENABLED=0 mise exec -- go -C "$root/temporal-host-tests" test -tags test_dep -count=1 -v ./...
for lane in a b; do
  test "$(git -C "$root/source-$lane" rev-parse HEAD)" = 8d429e97409b1537ecb347c359129bffafa63f78
  test -z "$(git -C "$root/source-$lane" status --porcelain --untracked-files=all)"
done
