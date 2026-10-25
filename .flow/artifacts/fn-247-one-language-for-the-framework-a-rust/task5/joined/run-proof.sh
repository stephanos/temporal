#!/usr/bin/env bash
set -euo pipefail
repo=/Users/stephan/Workspace/skunkworks/umpire/temporal
proof=$(cd "$(dirname "$0")" && pwd -P)
joined_sha=523d131d2cf992b04a43b4d595ea0f1a434f44b6
export BASH_ENV=/dev/null
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false CARGO_BUILD_JOBS=4
unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS RUSTC_WRAPPER RUSTC_WORKSPACE_WRAPPER RUSTC_BOOTSTRAP
run() {
  local name=$1
  local expected=$2
  shift 2
  printf '%q ' "$@" >> "$proof/commands.log"
  printf '\n' >> "$proof/commands.log"
  set +e
  "$@" > "$proof/$name.log" 2>&1
  local result=$?
  set -e
  printf '%s\n' "$result" > "$proof/$name.rc"
  printf '%s rc=%s expected=%s\n' "$name" "$result" "$expected" | tee -a "$proof/status.log"
  if [ "$result" -ne "$expected" ]; then tail -60 "$proof/$name.log"; exit "$result"; fi
}
cd "$repo"
printf '%s\n' "$joined_sha" > "$proof/source-commit"
date -u +%FT%TZ > "$proof/started-at"
uname -a > "$proof/platform.txt"
sha256sum "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm" > "$proof/old-main-asset.sha256"
run rustc-version 0 mise exec -- rustc -vV
run cargo-version 0 mise exec -- cargo -V
run go-version 0 mise exec -- go version
run protoc-version 0 mise exec -- protoc --version
run mise-version 0 mise --version
for name in source-a different-source-b; do
  run "clone-$name" 0 git clone --shared --no-checkout "$repo" "$proof/$name"
  run "checkout-$name" 0 git -C "$proof/$name" checkout --detach "$joined_sha"
  test -z "$(git -C "$proof/$name" status --porcelain)"
  test ! -e "$proof/clean-target-${name##*-}"
  git -C "$proof/$name" rev-parse HEAD >> "$proof/clone-source-shas"
  git -C "$proof/$name" ls-tree -r HEAD model > "$proof/$name-tree.txt"
done
source_a="$proof/source-a"
source_b="$proof/different-source-b"
target_a="$proof/clean-target-a"
target_b="$proof/clean-target-b"
test ! -e "$target_a"
test ! -e "$target_b"
run stock-a-build 0 env CARGO_TARGET_DIR="$target_a" bash "$source_a/model/scripts/guest.sh" umpire-stock-guest "$proof/stock-a.wasm"
run temporal-a-build 0 env CARGO_TARGET_DIR="$target_a" bash "$source_a/model/scripts/guest.sh" umpire-temporal-guest "$proof/temporal-a.wasm"
test ! -e "$target_b"
sha256sum "$proof/temporal-a.wasm" > "$proof/check-output-before.sha256"
run temporal-b-cold-freshness 0 env CARGO_TARGET_DIR="$target_b" bash "$source_b/model/scripts/guest-freshness.sh" umpire-temporal-guest "$proof/temporal-a.wasm"
sha256sum -c "$proof/check-output-before.sha256" > "$proof/check-output-preserved.log"
run stock-b-build 0 env CARGO_TARGET_DIR="$target_b" bash "$source_b/model/scripts/guest.sh" umpire-stock-guest "$proof/stock-b.wasm"
run temporal-b-publish-disposable 0 env CARGO_TARGET_DIR="$target_b" bash "$source_b/model/scripts/guest.sh" umpire-temporal-guest "$proof/temporal-b.wasm"
run compare-stock 0 cmp "$proof/stock-a.wasm" "$proof/stock-b.wasm"
run compare-temporal 0 cmp "$proof/temporal-a.wasm" "$proof/temporal-b.wasm"
run temporal-a-cached-freshness 0 env CARGO_TARGET_DIR="$target_a" bash "$source_a/model/scripts/guest-freshness.sh" umpire-temporal-guest "$proof/temporal-a.wasm"
run old-asset-stale-readonly 2 env CARGO_TARGET_DIR="$target_a" bash "$source_a/model/scripts/guest.sh" umpire-temporal-guest "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm" --check
sha256sum -c "$proof/old-main-asset.sha256" > "$proof/main-asset-preserved.log"
sha256sum "$proof/stock-a.wasm" "$proof/stock-b.wasm" "$proof/temporal-a.wasm" "$proof/temporal-b.wasm" > "$proof/new-guests.sha256"
wc -c "$proof/stock-a.wasm" "$proof/temporal-a.wasm" > "$proof/guest-sizes.txt"
stat -c '%d:%i %n' "$target_a" "$target_b" > "$proof/target-identities.txt"
for source in "$source_a" "$source_b"; do
  test -z "$(git -C "$source" status --porcelain)"
  git -C "$source" status --porcelain >> "$proof/clean-source-status.txt"
done
date -u +%FT%TZ > "$proof/finished-at"
printf 'PROOF_PASSED %s\n' "$proof" | tee -a "$proof/status.log"
