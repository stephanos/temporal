#!/bin/bash
# usage: D19_OUT=<scratch dir> fn105-d19-sweep.sh <name> <strict|forward> <suite> <suite|leaf> <source-dir> <seed-batch>...
# Runs fn105-d19-run.sh once per seed batch (bounding retained artifacts) and prints one line per seed:
# result, unfairness, transfers started at the first measured activity poll, and the failing source line.
set -euo pipefail
HERE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RUN=${D19_RUN:-$HERE/fn105-d19-run.sh}; TL=${D19_TIMELINE:-$HERE/fn105-d19-timeline.py}
OUT=${D19_OUT:?set D19_OUT to a directory inside the scratchpad}
[ "$#" -ge 6 ] || { echo "usage: $0 <name> <strict|forward> <suite> <suite|leaf> <source-dir> <seed-batch>..." >&2; exit 2; }
name=$1; tick=$2; suite=$3; sel=$4; dir=$5; shift 5
pat="^$suite\$"
if [ "$sel" = leaf ]; then pat="^$suite\$/^Test_Activity_Basic\$"; fi
for batch in "$@"; do
  n="$name-$(echo "$batch" | tr ',' '_')"
  mkdir -p "$OUT/out"
  "$RUN" "$n" "$batch" "$tick" "$pat" "$dir" > "$OUT/out/$n.summary" 2>&1 || true
  { grep -E "^name=|classification=" "$OUT/out/$n.summary" || true; } | sed 's/ output=.*//; s/ retained-success.*//'
  for s in $(echo "$batch" | tr ',' ' '); do
    o="$OUT/out/$n"
    if [ ! -f "$o/seed$s-stdout.gz" ]; then echo "seed=$s NO OUTPUT"; continue; fi
    res=$( { gzcat "$o/seed$s-stdout.gz" | grep -oE -- "--- (PASS|FAIL): $suite/Test_Activity_Basic \([0-9.]+s\)" || true; } | head -1)
    tl=$(python3 "$TL" --suite "$suite" "$o/seed$s-stderr.gz" "$o/seed$s-stdout.gz" 2>/dev/null || true)
    unf=$( { echo "$tl" | grep -oE "logged=[0-9.None]+" || true; } | head -1)
    tr=$( { echo "$tl" | grep -oE "transfer started=[0-9]+/225 from [0-9]+/15 workflows" || true; } | head -1)
    at=$( { gzcat "$o/seed$s-stdout.gz" | grep -A1 "Error Trace" | grep -oE "priority_fairness_test.go:[0-9]+" || true; } | tr '\n' ' ')
    echo "seed=$s $res unfairness $unf; at first poll: $tr; failed at: ${at:-none}"
  done
done
