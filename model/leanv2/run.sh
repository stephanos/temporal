#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
if [[ $# -gt 1 || ( $# -eq 1 && "$1" != --all ) ]]; then
  echo "usage: ./run.sh [--all]" >&2
  exit 2
fi
mkdir -p results.local
lake build DslExperimentTests dsl-experiment
.lake/build/bin/dsl-experiment > results.local/core-first.txt
.lake/build/bin/dsl-experiment > results.local/core-second.txt
cmp results.local/core-first.txt results.local/core-second.txt
cat results.local/core-first.txt
echo "deterministic_receipt=true"
if [[ ${1:-} == --all ]]; then
  ./run-baseline.sh
  ./veil/run-core.sh
  ./veil/run-edit.sh
fi
