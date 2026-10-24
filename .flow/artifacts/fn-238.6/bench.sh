#!/usr/bin/env bash
# fn-238.6 release benchmark: interpreted vs compiled Checker::explore_within on Kettle and
# the 2,001-start wide shape, interleaved, medians and ratio. Usage:
#   bash .flow/tmp/fn-238.6/bench.sh <worktree> [label] [scale]
set -euo pipefail
worktree=${1:?worktree}
label=${2:-run}
scale=${3:-1}
out=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-238.6
export CARGO_TARGET_DIR="$worktree/model/target"
cd "$worktree/model"
mise exec -- cargo test --release -p umpire-fixtures --test interpreted --no-run --quiet
{
  echo "commit $(git -C "$worktree" describe --always --dirty)  $(date -Is)  load: $(cut -d' ' -f1-3 /proc/loadavg)  nproc $(nproc)"
  git -C "$worktree" diff --stat HEAD
  UMPIRE_TIMING_SCALE="$scale" UMPIRE_TIMING_OUT="$out/bench-$label.txt" \
    mise exec -- cargo test --release -p umpire-fixtures --test interpreted --quiet -- --ignored --nocapture timing 2>&1 | grep -E "^(kettle|wide):|test result|Error|panicked"
  echo "load after: $(cut -d' ' -f1-3 /proc/loadavg)"
} | tee "$out/bench-$label.log"
