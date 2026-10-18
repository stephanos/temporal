#!/usr/bin/env bash
set -euo pipefail
here="$(cd -- "$(dirname -- "$0")" && pwd -P)"
root="$(cd -- "$here/../../.." && pwd -P)"
cd -- "$root"
logs="$root/.flow/tmp/fn1941"
mkdir -p -- "$logs"
inputs=("$here/project.scala" model/OutputPaths.scala)
for directory in dsl realize checker; do
  for file in "model/framework/$directory/"*.scala; do
    case "$file" in *.test.scala | */project.scala) ;; *) inputs+=("$file") ;; esac
  done
done
if [[ "${1:-test}" == refusals ]]; then
  for fixture in OwnedVocabulary FiniteOnly; do
    scratch="$(mktemp -d "$logs/$fixture.XXXXXX")"
    cp -- "$here/refused/$fixture.scala.txt" "$scratch/Bank.scala"
    set +e
    mise exec -- scala-cli compile --server=false --workspace "$scratch/workspace" "${inputs[@]}" "$scratch/Bank.scala" >"$logs/$fixture.log" 2>&1
    status=$?
    set -e
    if [[ "$status" == 0 ]]; then
      printf '%s unexpectedly compiled\n' "$fixture" >&2
      exit 1
    fi
    grep -F 'has no IR schema' "$logs/$fixture.log" >/dev/null
    if [[ "$fixture" == OwnedVocabulary ]]; then
      grep -F 'cannot establish a reference' "$logs/$fixture.log" >/dev/null
    fi
    printf '%s: expected schema refusal (compiler exit %s)\n' "$fixture" "$status"
  done
  exit 0
fi
[[ "${1:-test}" == test ]] || { printf 'usage: bash run.sh [test|refusals]\n' >&2; exit 2; }
inputs+=("$here/Bank.scala" "$here/Bank.test.scala")
set +e
mise exec -- scala-cli test --server=false "${inputs[@]}" --require-tests --test-only 'parametricprototype.ParametricBankTest' >"$logs/prototype.log" 2>&1
status=$?
set -e
printf 'prototype_rc=%s\n' "$status"
tail -30 "$logs/prototype.log"
exit "$status"
