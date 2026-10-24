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
  if CLIPPY_CONF_DIR="$PWD/umpire/$directory" mise exec -- clippy-driver --edition=2024 \
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
