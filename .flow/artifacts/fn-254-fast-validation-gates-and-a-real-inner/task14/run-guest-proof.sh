#!/bin/bash
set -euo pipefail
unset BASH_ENV RUSTFLAGS CARGO_ENCODED_RUSTFLAGS RUSTDOCFLAGS RUSTC_WRAPPER RUSTC_WORKSPACE_WRAPPER CARGO_BUILD_TARGET RUSTC_BOOTSTRAP
for profile_var in ${!CARGO_PROFILE_@}; do unset "$profile_var"; done
cd /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn25414-probes
[ "$(git rev-parse HEAD)" = 5ba482b5b73ca4d85ab758568d9a25ec4c4126b4 ]
[ -z "$(git status --porcelain)" ]
export CARGO_BUILD_JOBS=2 RUST_TEST_THREADS=2 CARGO_INCREMENTAL=0 MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
proof=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn254-task14
cache=/home/agent/.cache/codex-build/umpire-milestones-20261024/fn25414-probes
sha256sum model/temporal/hosts/go/umpire_temporal_guest.wasm > "$proof/guest-old.sha256"
(cd model/umpire && mise exec -- rustc -Vv && mise exec -- cargo -V && mise --version) > "$proof/tool-manifest.txt"
for lane in a b; do
  export CARGO_TARGET_DIR="$cache/reproduction-$lane"
  [ ! -e "$CARGO_TARGET_DIR" ]
  bash model/scripts/guest.sh umpire-temporal-guest "$proof/temporal-$lane.wasm" > "$proof/guest-$lane.log" 2>&1
  sha256sum "$CARGO_TARGET_DIR/xtask-bootstrap/debug/umpire-guest-driver" > "$proof/native-driver-$lane.sha256"
  find "$CARGO_TARGET_DIR/guest-cache" -mindepth 1 -maxdepth 1 -type d > "$proof/guest-cache-$lane.txt"
  sha256sum "$proof/temporal-$lane.wasm" > "$proof/guest-$lane.sha256"
done
cmp "$proof/temporal-a.wasm" "$proof/temporal-b.wasm"
if cmp model/temporal/hosts/go/umpire_temporal_guest.wasm "$proof/temporal-a.wasm"; then
  printf 'unchanged\n' > "$proof/publication-status.txt"
else
  printf 'changed\n' > "$proof/publication-status.txt"
fi
