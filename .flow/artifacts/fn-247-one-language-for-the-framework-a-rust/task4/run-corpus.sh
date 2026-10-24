#!/usr/bin/env bash
set -euo pipefail
repo=$(cd -- "$(dirname -- "$0")/../../../.." && pwd -P)
evidence="$repo/.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task4"
scratch="$repo/.flow/tmp/fn2474-corpus"
mkdir -p "$scratch"
cd "$repo/model"
mise exec -- cargo build -p umpire-backend-process -p umpire-tla --locked --message-format=json > "$evidence/corpus-build.jsonl" 2> "$evidence/corpus-build.stderr"
for owner in direct compat; do
  if [ "$owner" = direct ]; then crate=umpire_backend_process; else crate=umpire_tla; fi
  rlib=$(jq -r --arg crate "$crate" 'select(.reason == "compiler-artifact" and .target.name == $crate) | .filenames[] | select(endswith(".rlib"))' "$evidence/corpus-build.jsonl")
  test -n "$rlib"
  mise exec -- rustc --edition=2024 "$evidence/$owner-corpus.rs" --extern "$crate=$rlib" -L "dependency=$(dirname -- "$rlib")/deps" -o "$scratch/$owner-corpus"
  cd "$repo"
  "$scratch/$owner-corpus" > "$evidence/$owner-corpus.stdout" 2> "$evidence/$owner-corpus.stderr"
  cmp "$evidence/../task1/baseline-corpus.stdout" "$evidence/$owner-corpus.stdout"
  cmp "$evidence/../task1/baseline-corpus.stderr" "$evidence/$owner-corpus.stderr"
  cd "$repo/model"
done
printf 'direct and compatibility API corpus stdout/stderr match retained baseline exactly\n'
