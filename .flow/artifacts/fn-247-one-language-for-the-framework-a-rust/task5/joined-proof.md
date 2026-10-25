# Joined guest proof instructions

Run after committing the merged task 5/6/7 inputs. Capture the command transcript, statuses, source SHA, tool versions and target identities. Re-run focused xtask controls after resolving dispatcher, dependency or Make conflicts. Preserve the existing canonical asset's old hash before publishing. Use fresh clone/target names for the two-build proof.

```bash
set -euo pipefail
repo=$(pwd -P)
joined_sha=$(git rev-parse HEAD)
proof=$(mktemp -d "$repo/.flow/tmp/fn2475-joined-proof.XXXXXX")
printf '%s\n' "$joined_sha" > "$proof/source-commit"
sha256sum "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm" > "$proof/old-asset.sha256"
for name in source-a different-source-b; do
  git clone --shared --no-checkout "$repo" "$proof/$name"
  git -C "$proof/$name" checkout --detach "$joined_sha"
  test -z "$(git -C "$proof/$name" status --porcelain)"
done
for pair in source-a:a different-source-b:b; do
  source="$proof/${pair%:*}"
  side=${pair#*:}
  target="$proof/clean-target-$side"
  test ! -e "$target"
  CARGO_TARGET_DIR="$target" bash "$source/model/scripts/guest.sh" umpire-stock-guest "$proof/stock-$side.wasm"
  CARGO_TARGET_DIR="$target" bash "$source/model/scripts/guest.sh" umpire-temporal-guest "$proof/temporal-$side.wasm"
  test -z "$(git -C "$source" status --porcelain)"
done
cmp "$proof/stock-a.wasm" "$proof/stock-b.wasm"
cmp "$proof/temporal-a.wasm" "$proof/temporal-b.wasm"
sha256sum "$proof/stock-a.wasm" "$proof/stock-b.wasm" "$proof/temporal-a.wasm" "$proof/temporal-b.wasm" > "$proof/new-guests.sha256"
wc -c "$proof/stock-a.wasm" "$proof/temporal-a.wasm"
```

After both pairs agree, the conductor publishes the one proven Temporal output and records the cause as native identity schema/compiler-input migration. Keep scientific golden expectations. Any additional pin/golden change needs its actual semantic cause recorded. The following commands intentionally modify the committed asset and therefore belong to the conductor's joined checkout.

```bash
cp "$proof/temporal-a.wasm" "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm"
sha256sum "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm" > "$proof/published-asset.sha256"
CARGO_TARGET_DIR="$proof/clean-target-a" bash "$repo/model/scripts/guest-freshness.sh" umpire-temporal-guest "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm"
CARGO_TARGET_DIR="$proof/clean-target-b" bash "$proof/different-source-b/model/scripts/guest.sh" umpire-temporal-guest "$repo/model/temporal/hosts/go/umpire_temporal_guest.wasm" --check
```

The first freshness invocation validates the actual pinned prerequisites. The second independently checks the second clean build against the published asset. For the actual cold native bootstrap path, keep the initial bootstrap target absent and use the shell freshness invocation after publication; the lane's fake controls already cover admission failures. If target a is reused above, select a fresh bootstrap-only parent for this cold proof and document the extra guest build.

After publication, run the scoped Temporal host suite against the new embedded asset.

```bash
CGO_ENABLED=0 mise exec -- go -C "$repo/model/temporal/hosts/go" test -tags test_dep -count=1 ./...
```

Before completion, retain joined R18 package/features and guest-closure checks plus semantic host parity. Task 3's one `UMPIRE_APALACHE=require make model-validate` executes all generic, Kettle and Temporal Go host suites and the six guest controls, then its edited-checker benchmark. Reuse that covering result instead of running a second full gate here. Record current source/target paths and actual outcomes; these instructions alone discharge no proof.
