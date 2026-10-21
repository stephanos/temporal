#!/usr/bin/env bash
set -euxo pipefail
repository=$1
root=$2
pin=8d429e97409b1537ecb347c359129bffafa63f78
test ! -e "$root"
mkdir -p "$root"
date -u
uname -a
df -h "$root"
export CARGO_BUILD_JOBS=2
unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS CARGO_TARGET_DIR
for lane in a b; do
  source="$root/source-$lane"
  target="$root/target-$lane"
  test ! -e "$source"
  test ! -e "$target"
  git clone --shared --no-checkout "$repository" "$source"
  git -C "$source" checkout --detach "$pin"
  test "$(git -C "$source" rev-parse HEAD)" = "$pin"
  test -z "$(git -C "$source" status --porcelain --untracked-files=all)"
  cd "$source"
  mise trust "$source/mise.toml"
  mise exec -- rustc -Vv
  mise exec -- cargo -V
  mise exec -- go version
  mise exec -- protoc --version
  mise exec -- rustup component list --installed
  mise exec -- rustup target list --installed
  export CARGO_TARGET_DIR="$target"
  bash -n model/scripts/guest.sh
  bash model/scripts/guest.sh umpire-stock-guest "$root/stock-$lane.wasm"
  bash model/scripts/guest.sh umpire-temporal-guest "$root/temporal-$lane.wasm"
  test -z "$(git status --porcelain --untracked-files=all)"
done
cmp "$root/stock-a.wasm" "$root/stock-b.wasm"
cmp "$root/temporal-a.wasm" "$root/temporal-b.wasm"
python3 - "$root" <<'PY'
import hashlib
import json
from pathlib import Path
import sys

root = Path(sys.argv[1])
manifest = {}
for path in sorted(root.glob("*.wasm")):
    data = path.read_bytes()
    manifest[path.name] = {"bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}
for lane in ("a", "b"):
    path = root / f"target-{lane}"
    stat = path.stat()
    manifest[path.name] = {"path": str(path.resolve()), "device": stat.st_dev, "inode": stat.st_ino}
assert not (root / "target-a").samefile(root / "target-b")
result = json.dumps(manifest, indent=2) + "\n"
(root / "manifest.json").write_text(result)
print(result)
PY
cd "$root/source-a"
export CARGO_TARGET_DIR="$root/target-a"
bash model/scripts/tests/build-guests.sh
cmp "$root/stock-a.wasm" "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_stock_guest.wasm"
cmp "$root/temporal-a.wasm" "$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_temporal_guest.wasm"
cmp "$root/temporal-a.wasm" model/temporal/hosts/go/umpire_temporal_guest.wasm
export UMPIRE_WASM="$root/stock-a.wasm"
export UMPIRE_STATEFUL_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/stateful_control.wasm"
export UMPIRE_CAUGHT_PANIC_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/caught_panic_control.wasm"
export UMPIRE_OVERFLOW_WASM="$CARGO_TARGET_DIR/wasm32-wasip1/guest/overflow_control.wasm"
export UMPIRE_PANIC_NATIVE="$CARGO_TARGET_DIR/panic-native"
bash model/scripts/tests/guest-parity.sh
CGO_ENABLED=0 mise exec -- go -C model/temporal/hosts/go test -tags test_dep -count=1 -v ./...
for lane in a b; do
  test "$(git -C "$root/source-$lane" rev-parse HEAD)" = "$pin"
  test -z "$(git -C "$root/source-$lane" status --porcelain --untracked-files=all)"
done
date -u
df -h "$root"
