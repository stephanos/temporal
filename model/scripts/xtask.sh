#!/usr/bin/env bash
set -euo pipefail
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
cd "$(dirname "$0")/.."
if ! command -v mise >/dev/null 2>&1; then
  printf 'xtask bootstrap requires mise and installed pinned Rust/Cargo\n' >&2
  exit 1
fi
target=${CARGO_TARGET_DIR:-$PWD/target}
mkdir -p "$target"
target=$(cd "$target" && pwd -P)
bootstrap="$target/xtask-bootstrap"
if ! (
  unset RUSTC_WRAPPER RUSTC_WORKSPACE_WRAPPER RUSTFLAGS CARGO_ENCODED_RUSTFLAGS RUSTC_BOOTSTRAP
  mise exec -- cargo build --locked -p xtask --bin xtask --profile dev --target-dir "$bootstrap" >&2
); then
  printf 'xtask bootstrap failed: installed pinned Rust/Cargo are required\n' >&2
  exit 1
fi
if [ ! -x "$bootstrap/debug/xtask" ]; then
  printf 'xtask bootstrap did not produce an executable\n' >&2
  exit 1
fi
printf '%s\n' "$bootstrap/debug/xtask"
