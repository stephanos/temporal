#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
scratch=$(mktemp -d)
trap 'rm -rf "${scratch:?}"' EXIT
mkdir -p "$scratch/framework/scripts" "$scratch/caller" "$scratch/bin"
cp scripts/{xtask,guest,guest-freshness}.sh "$scratch/framework/scripts/"
cat > "$scratch/bin/mise" <<'TOOL'
#!/usr/bin/env bash
set -euo pipefail
[ "$MISE_AUTO_INSTALL" = false ] && [ "$MISE_EXEC_AUTO_INSTALL" = false ]
[ -z "${RUSTC_WRAPPER:-}" ] && [ -z "${RUSTFLAGS:-}" ]
[ -z "${CARGO_BUILD_TARGET:-}" ]
[ "$UMPIRE_TOOL_ROOT" = "$CONTROL_FRAMEWORK" ]
[ "$PWD" = "$CONTROL_FRAMEWORK" ]
[ "$1" = exec ] && [ "$2" = -- ] && [ "$3" = cargo ] && [ "$4" = build ]
shift 4
while [ "$#" -gt 0 ]; do
  case "$1" in --target-dir) shift; target=$1 ;; esac
  shift
done
mkdir -p "$target/debug"
cat > "$target/debug/xtask" <<'DRIVER'
#!/usr/bin/env bash
set -euo pipefail
[ "$PWD" = "$CONTROL_CALLER" ]
[ "$UMPIRE_TOOL_ROOT" = "$CONTROL_FRAMEWORK" ]
printf '%s\n' "$@" > "$CONTROL_OUTPUT"
DRIVER
cp "$target/debug/xtask" "$target/debug/umpire-guest-driver"
chmod +x "$target/debug/"*
TOOL
chmod +x "$scratch/bin/mise"
export PATH="$scratch/bin:$PATH" CONTROL_FRAMEWORK="$scratch/framework" CONTROL_CALLER="$scratch/caller" CONTROL_OUTPUT="$scratch/arguments"
cd "$scratch/caller"
CARGO_BUILD_TARGET=wasm32-wasip1 CARGO_TARGET_DIR=target RUSTC_WRAPPER=bad RUSTFLAGS=bad bash "$scratch/framework/scripts/guest.sh" example relative.wasm --check
[ -x "$scratch/caller/target/xtask-bootstrap/debug/umpire-guest-driver" ]
printf 'guest\nexample\n%s/relative.wasm\n--check\n' "$scratch/caller" > "$scratch/expected"
cmp "$scratch/expected" "$scratch/arguments"
CARGO_TARGET_DIR=target bash "$scratch/framework/scripts/guest-freshness.sh" example relative.wasm
printf 'guest-freshness\nexample\n%s/relative.wasm\n' "$scratch/caller" > "$scratch/expected"
cmp "$scratch/expected" "$scratch/arguments"
printf 'bootstrap controls: local tool root, caller cwd/output/target and two native executables passed (fake build)\n'
