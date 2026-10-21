#!/usr/bin/env bash
set -euxo pipefail
repository=$1
system=$2
root="$repository/.flow/tmp/fn232-diagnosis"
source="$repository/.flow/tmp/fn232-certification-$system/source-a"
export CARGO_TARGET_DIR="$root/target-$system"
export CARGO_BUILD_JOBS=2
export RUSTC_WRAPPER="$root/rustc-log.py"
export FN232_RUSTC_LOG="$root/rustc-$system.jsonl"
unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS
cd "$source"
test ! -e "$CARGO_TARGET_DIR"
test "$(git rev-parse HEAD)" = 8d429e97409b1537ecb347c359129bffafa63f78
test -z "$(git status --porcelain --untracked-files=all)"
date -u
uname -a
bash model/scripts/guest.sh umpire-stock-guest "$root/stock-$system.wasm"
cmp "$root/stock-$system.wasm" "$repository/.flow/tmp/fn232-certification-$system/stock-a.wasm"
test -z "$(git status --porcelain --untracked-files=all)"
date -u
