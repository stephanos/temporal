#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -lt 2 ] || [ "$#" -gt 3 ] || { [ "$#" -eq 3 ] && [ "$3" != --check ]; }; then
  printf 'usage: guest.sh <crate> <output> [--check]\n' >&2
  exit 1
fi
crate=$1
case "$2" in
  /*) output=$2 ;;
  *) output="$PWD/$2" ;;
esac
mode=${3:-build}
cd "$(dirname "$0")/.."
workspace=$(pwd -P)
sysroot=$(mise exec -- rustc --print sysroot) || exit 1
cargo_home=${CARGO_HOME:-$HOME/.cargo}
target=${CARGO_TARGET_DIR:-$workspace/target}
mkdir -p "$target"
export CARGO_TARGET_DIR
CARGO_TARGET_DIR=$(cd "$target" && pwd -P)
flags=${CARGO_ENCODED_RUSTFLAGS:-}
if [ -z "$flags" ] && [ -n "${RUSTFLAGS:-}" ]; then
  read -r -a rustflags <<< "$RUSTFLAGS"
  printf -v flags '%s\x1f' "${rustflags[@]}"
  flags=${flags%$'\x1f'}
fi
for mapping in "$workspace=/workspace" "$cargo_home/registry=/cargo/registry" "$sysroot=/rust/sysroot" "$CARGO_TARGET_DIR=/target"; do
  flags="${flags:+$flags$'\x1f'}--remap-path-prefix=$mapping"
done
artifacts=$(mktemp)
trap 'rm -f "$artifacts"' EXIT
if ! CARGO_ENCODED_RUSTFLAGS="$flags" mise exec -- cargo build -p "$crate" --lib \
  --profile guest --target wasm32-wasip1 --locked --message-format=json-render-diagnostics >"$artifacts"; then
  exit 1
fi
python3 - "$artifacts" "$output" "$mode" <<'PY'
import json
from pathlib import Path
import shutil
import sys

try:
    artifacts, output, mode = sys.argv[1:]
    candidates = [Path(filename)
                  for line in Path(artifacts).read_text().splitlines()
                  for item in [json.loads(line)]
                  if item.get("reason") == "compiler-artifact"
                  and "cdylib" in item["target"]["crate_types"]
                  for filename in item["filenames"] if filename.endswith(".wasm")]
    if len(candidates) != 1:
        raise ValueError(f"expected one guest wasm artifact, found {len(candidates)}")
    artifact = candidates[0]
    output = Path(output)
    if mode == "--check":
        built = artifact.read_bytes()
        existing = output.read_bytes() if output.exists() else None
        if existing != built:
            size = "missing" if existing is None else f"{len(existing)} bytes"
            offset = next((i for i, (a, b) in enumerate(zip(existing or b"", built)) if a != b),
                          min(len(existing or b""), len(built)))
            print(f"stale guest {output}: existing={size}, built={len(built)} bytes; "
                  f"first differing offset={offset} (zero-based)", file=sys.stderr)
            sys.exit(2)
    else:
        output.parent.mkdir(parents=True, exist_ok=True)
        if artifact.resolve() != output.resolve():
            shutil.copyfile(artifact, output)
except (OSError, ValueError, KeyError) as error:
    print(f"guest build failed: {error}", file=sys.stderr)
    sys.exit(1)
PY
