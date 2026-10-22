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
target=$(cd "$target" && pwd -P)
wrapper="$workspace/scripts/guest-rustc.py"
export UMPIRE_GUEST_COMPILER
UMPIRE_GUEST_COMPILER=$(mise exec -- rustc -V) || exit 1
cache_key=$(python3 - "$wrapper" "$workspace/Cargo.lock" "$UMPIRE_GUEST_COMPILER" <<'PY'
import hashlib
import json
from pathlib import Path
import sys

inputs = [hashlib.sha256(Path(path).read_bytes()).hexdigest() for path in sys.argv[1:3]]
inputs.append(sys.argv[3])
print(hashlib.sha256(json.dumps(inputs).encode()).hexdigest()[:16])
PY
)
export CARGO_TARGET_DIR="$target/guest-cache/$cache_key"
launcher="$target/guest-wrappers/$cache_key/guest-rustc.py"
python3 - "$wrapper" "$launcher" <<'PY'
from pathlib import Path
import os
import sys

if any(character.isspace() for character in sys.executable):
    sys.exit("guest build requires a Python executable path without whitespace")
source, launcher = map(Path, sys.argv[1:])
# /usr/bin/env clears Cargo's dynamic-library environment on macOS.
content = b"#!" + os.fsencode(sys.executable) + b"\n" + source.read_bytes().split(b"\n", 1)[1]
launcher.parent.mkdir(parents=True, exist_ok=True)
temporary = launcher.with_name(launcher.name + "." + str(os.getpid()))
temporary.write_bytes(content)
temporary.chmod(0o755)
temporary.replace(launcher)
PY
wrapper=$launcher
export UMPIRE_GUEST_WORKSPACE="$workspace" UMPIRE_GUEST_SYSROOT="$sysroot"
export UMPIRE_GUEST_CARGO_HOME="$cargo_home" UMPIRE_GUEST_TARGET="$CARGO_TARGET_DIR"
export UMPIRE_GUEST_LOCK="$CARGO_TARGET_DIR/umpire-lock.json"
mkdir -p "$CARGO_TARGET_DIR"
mise exec -- cargo metadata --format-version 1 --locked \
  | python3 "$workspace/scripts/guest-lock.py" "$workspace/Cargo.lock" "$UMPIRE_GUEST_LOCK" || exit 1
if [ "${RUSTC_WRAPPER:-}" != "$wrapper" ]; then
  export UMPIRE_GUEST_PREVIOUS_WRAPPER="${RUSTC_WRAPPER:-}"
fi
export RUSTC_WRAPPER="$wrapper"
flags=${CARGO_ENCODED_RUSTFLAGS:-}
if [ -z "$flags" ] && [ -n "${RUSTFLAGS:-}" ]; then
  read -r -a rustflags <<< "$RUSTFLAGS"
  printf -v flags '%s\x1f' "${rustflags[@]}"
  flags=${flags%$'\x1f'}
fi
for mapping in "$workspace=/workspace" "$cargo_home/registry=/cargo/registry" "$sysroot=/rust/sysroot" "$CARGO_TARGET_DIR=/target"; do
  flags="${flags:+$flags$'\x1f'}--remap-path-prefix=$mapping"
done
for flag in -Cpanic=unwind -Ctarget-feature=+exception-handling -Cllvm-args=-wasm-use-legacy-eh=false; do
  flags="${flags:+$flags$'\x1f'}$flag"
done
artifacts=$(mktemp)
trap 'rm -f "$artifacts"' EXIT
if ! RUSTC_BOOTSTRAP=1 CARGO_ENCODED_RUSTFLAGS="$flags" mise exec -- cargo build -p "$crate" --lib \
  --profile guest --target wasm32-wasip1 -Zbuild-std=std,panic_unwind --locked --message-format=json-render-diagnostics >"$artifacts"; then
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
