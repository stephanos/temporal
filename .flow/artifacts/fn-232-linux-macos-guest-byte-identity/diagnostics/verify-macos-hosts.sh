#!/usr/bin/env bash
set -euxo pipefail
repository=$1
root="$repository/.flow/tmp/fn232-diagnosis"
proof="$repository/.flow/tmp/fn232-certification-macos"
source="$proof/source-a"
export UMPIRE_WASM="$root/stock-macos-replay-paths.wasm"
export UMPIRE_STATEFUL_WASM="$proof/target-a/wasm32-wasip1/guest/stateful_control.wasm"
export UMPIRE_CAUGHT_PANIC_WASM="$proof/target-a/wasm32-wasip1/guest/caught_panic_control.wasm"
export UMPIRE_OVERFLOW_WASM="$proof/target-a/wasm32-wasip1/guest/overflow_control.wasm"
export UMPIRE_PANIC_NATIVE="$proof/target-a/panic-native"
cd "$source"
bash model/scripts/tests/guest-parity.sh
cmp "$root/temporal-macos-replay-paths.wasm" model/temporal/hosts/go/umpire_temporal_guest.wasm
CGO_ENABLED=0 mise exec -- go -C model/temporal/hosts/go test -tags test_dep -count=1 ./...
test -z "$(git status --porcelain --untracked-files=all)"
