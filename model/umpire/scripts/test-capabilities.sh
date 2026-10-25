#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT

cat_source='#![allow(dead_code)]
use std::fs::read as aliased_read;
pub fn forbidden() {
    let _ = aliased_read("unused");
    let _ = std::time::Instant::now();
    let _ = std::thread::spawn(|| ());
    let _ = std::path::Path::new("unused").exists();
    std::thread::yield_now();
    let _ = std::fs::DirBuilder::new();
    let _: Option<std::fs::File> = None;
    let _: Option<std::time::SystemTime> = None;
    let _: Option<std::thread::JoinHandle<()>> = None;
}
'
printf '%s' "$cat_source" > "$scratch/control.rs"
for pair in dsl:author/dsl checker:backends/native; do
  core=${pair%%:*}
  directory=${pair#*:}
  if CLIPPY_CONF_DIR="$PWD/$directory" mise exec -- clippy-driver --edition=2024 \
    --crate-type=lib --emit=metadata --out-dir "$scratch" "$scratch/control.rs" \
    -D clippy::disallowed_methods -D clippy::disallowed_types > "$scratch/$core.log" 2>&1; then
    printf 'capability negative control unexpectedly compiled: %s\n' "$core" >&2
    exit 1
  fi
  for symbol in 'std::fs::read' 'std::time::Instant::now' 'std::thread::spawn' \
    'std::fs::File' 'std::time::SystemTime' 'std::thread::JoinHandle' \
    'std::path::Path::exists' 'std::thread::yield_now' 'std::fs::DirBuilder::new'; do
    if ! grep -F "$symbol" "$scratch/$core.log" >/dev/null; then
      cat "$scratch/$core.log" >&2
      printf 'missing capability refusal: %s (%s)\n' "$symbol" "$core" >&2
      exit 1
    fi
  done
done
printf 'capability controls: both core policies reject filesystem, clock and thread APIs/types\n'

for root in author/dsl/src/lib.rs dev/xtask/src/main.rs dev/guest-driver/src/main.rs \
  backends/native/build.rs author/dsl/tests/main.rs; do
  policy=$(grep -Fx '#![forbid(unsafe_code)]' "$root")
  printf '%s\n#[allow(unsafe_code)]\npub unsafe fn forbidden() {}\n' "$policy" > "$scratch/unsafe.rs"
  if mise exec -- rustc --edition=2024 --crate-type=lib --emit=metadata \
    --out-dir "$scratch" "$scratch/unsafe.rs" > "$scratch/unsafe.log" 2>&1; then
    printf 'unsafe override unexpectedly compiled: %s\n' "$root" >&2
    exit 1
  fi
  grep -F 'E0453' "$scratch/unsafe.log" >/dev/null || { cat "$scratch/unsafe.log" >&2; exit 1; }
done
printf '#![deny(unsafe_code)]\n#[allow(unsafe_code)]\npub unsafe fn audited() {}\n' > "$scratch/unsafe.rs"
mise exec -- rustc --edition=2024 --crate-type=lib --emit=metadata --out-dir "$scratch" "$scratch/unsafe.rs"
printf '#![deny(unsafe_code)]\npub unsafe fn forbidden() {}\n' > "$scratch/unsafe.rs"
if mise exec -- rustc --edition=2024 --crate-type=lib --emit=metadata \
  --out-dir "$scratch" "$scratch/unsafe.rs" > "$scratch/unsafe.log" 2>&1; then
  printf 'unaudited unsafe unexpectedly compiled\n' >&2
  exit 1
fi
grep -F 'declaration of an `unsafe` function' "$scratch/unsafe.log" >/dev/null || { cat "$scratch/unsafe.log" >&2; exit 1; }
printf 'unsafe controls: safe target roots reject overrides; deny requires a local allowance\n'
