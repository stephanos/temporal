#!/usr/bin/env bash
set -euo pipefail
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
case "${CARGO_TARGET_DIR:-}" in
  ""|/*) ;;
  *) export CARGO_TARGET_DIR="$PWD/$CARGO_TARGET_DIR" ;;
esac
cd "$(dirname "$0")/.."
export UMPIRE_TOOL_ROOT="$PWD"
if ! command -v mise >/dev/null 2>&1; then
  printf 'xtask bootstrap requires mise and installed pinned Rust/Cargo\n' >&2
  exit 1
fi
target=${CARGO_TARGET_DIR:-$PWD/target}
mkdir -p "$target"
target=$(cd "$target" && pwd -P)
bootstrap="$target/xtask-bootstrap"
if ! (
  unset CARGO_BUILD_TARGET RUSTC_WRAPPER RUSTC_WORKSPACE_WRAPPER RUSTFLAGS CARGO_ENCODED_RUSTFLAGS RUSTC_BOOTSTRAP
  mise exec -- cargo build --locked -p xtask --bin xtask -p umpire-guest-driver --bin umpire-guest-driver --profile dev --target-dir "$bootstrap" >&2
); then
  printf 'xtask bootstrap failed: installed pinned Rust/Cargo are required\n' >&2
  exit 1
fi
if [ ! -x "$bootstrap/debug/xtask" ] || [ ! -x "$bootstrap/debug/umpire-guest-driver" ]; then
  printf 'xtask bootstrap did not produce an executable\n' >&2
  exit 1
fi
printf '%s\n' "$bootstrap/debug/xtask"
