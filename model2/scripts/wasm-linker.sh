#!/usr/bin/env bash
set -euo pipefail
script_dir=$(cd "$(dirname "$0")" && pwd -P)
sysroot=$(rustc --print sysroot)
host=$(rustc -vV | sed -n 's/^host: //p')
"$sysroot/lib/rustlib/$host/bin/rust-lld" "$@"
python3 "$script_dir/wasm-export.py" "$@"
