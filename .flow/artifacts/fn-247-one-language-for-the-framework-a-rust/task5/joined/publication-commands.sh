#!/usr/bin/env bash
set -euo pipefail
repo=/Users/stephan/Workspace/skunkworks/umpire/temporal
proof=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2475-joined-proof.MstS6i
# The conductor runs this after releasing the immutable review window.
cd "$repo"
test "$(sha256sum "$proof/temporal-a.wasm" | cut -d ' ' -f 1)" = bc27af7d5b6b1ed8fa5c524015046f3a8b7c12d2d488e9ce7cb3df3ea9217872
cp "$proof/temporal-a.wasm" "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm"
sha256sum "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm"
CARGO_TARGET_DIR="$proof/clean-target-a" BASH_ENV=/dev/null bash "$proof/source-a/model/scripts/guest-freshness.sh" umpire-temporal-guest "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm"
CARGO_TARGET_DIR="$proof/clean-target-b" BASH_ENV=/dev/null bash "$proof/different-source-b/model/scripts/guest.sh" umpire-temporal-guest "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm" --check
