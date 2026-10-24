#!/usr/bin/env bash
set -euo pipefail
repo=$(cd -- "$(dirname -- "$0")/../../../.." && pwd -P)
evidence="$repo/.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task4"
target="$repo/.flow/tmp/fn2474-lazy-consumer-target"
cd "$repo/model"
RUSTFLAGS="-C link-arg=-Wl,-z,lazy" mise exec -- cargo build --offline --locked --manifest-path "$evidence/cold-consumer/Cargo.toml" --target-dir "$target" > "$evidence/cold-post-build.log" 2>&1
cd "$repo"
readelf -d "$target/debug/process-cold-consumer" > "$evidence/cold-post-dynamic.log"
if rg -q BIND_NOW "$evidence/cold-post-dynamic.log"; then exit 1; fi
test ! -e "$target/debug/umpire-process-fixture"
PATH=/no-runtime-tools "$target/debug/process-cold-consumer" "$repo" > "$evidence/cold-post-run.log" 2>&1
cat "$evidence/cold-post-run.log"
