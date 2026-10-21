#!/usr/bin/env bash
set -euxo pipefail
repository=$1
platform=$2
proof="$repository/.flow/tmp/fn236-snapshot-proof"
evidence="$repository/.flow/artifacts/fn-236-host-independent-guest-compiler-inputs"
root="$proof/$platform"
export CARGO_BUILD_JOBS=2
unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS CARGO_TARGET_DIR RUSTC_WRAPPER
date -u
uname -a
for lane in a b; do
  source="$root/source-$lane"
  target="$root/target-$lane"
  test ! -e "$target"
  cd "$source"
  mise trust "$source/mise.toml"
  mise exec -- rustc -Vv
  mise exec -- cargo -V
  mise exec -- go version
  mise exec -- protoc --version
  git rev-parse HEAD
  git status --short
  python3 "$evidence/source-manifest.py" "$source" > "$evidence/$platform-$lane-before.json"
  cmp "$proof/frozen-source.json" "$evidence/$platform-$lane-before.json"
  export CARGO_TARGET_DIR="$target"
  bash model/scripts/guest.sh umpire-stock-guest "$root/stock-$lane.wasm"
  bash model/scripts/guest.sh umpire-temporal-guest "$root/temporal-$lane.wasm"
  cmp "$root/temporal-$lane.wasm" model/temporal/hosts/go/umpire_temporal_guest.wasm
  python3 "$evidence/source-manifest.py" "$source" > "$evidence/$platform-$lane-after.json"
  cmp "$proof/frozen-source.json" "$evidence/$platform-$lane-after.json"
done
cmp "$root/stock-a.wasm" "$root/stock-b.wasm"
cmp "$root/temporal-a.wasm" "$root/temporal-b.wasm"
cd "$root/source-a"
export CARGO_TARGET_DIR="$root/target-a"
bash model/scripts/tests/build-guests.sh
cmp "$root/stock-a.wasm" "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_stock_guest.wasm"
cmp "$root/temporal-a.wasm" "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_temporal_guest.wasm"
export UMPIRE_WASM="$root/stock-a.wasm"
export UMPIRE_STATEFUL_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/stateful_control.wasm"
export UMPIRE_CAUGHT_PANIC_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/caught_panic_control.wasm"
export UMPIRE_OVERFLOW_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/overflow_control.wasm"
export UMPIRE_PANIC_NATIVE="$CARGO_TARGET_DIR/panic-native"
bash model/scripts/tests/guest-parity.sh
CGO_ENABLED=0 mise exec -- go -C model/temporal/hosts/go test -tags test_dep -count=1 -v ./...
python3 "$evidence/source-manifest.py" "$root/source-a" > "$evidence/$platform-a-final.json"
cmp "$proof/frozen-source.json" "$evidence/$platform-a-final.json"
date -u
