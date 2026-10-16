#!/usr/bin/env bash
# fn-182 R4 matched baseline/candidate measurement (conductor reserves the host).
# Usage: measure-r4.sh <baseline-worktree> <candidate-worktree> <evidence-root> [reps=3] [workloads="onebox multi-instance"]
# Each rep runs the existing qualification success cell in a fresh evidence root (cold Go
# preparation, cold engine compile on execution 1, warm cache hits afterwards), alternating
# ABBA order. Both helpers must be built in their own worktree beforehand (make -C tools/gomad wasm-helper).
set -euo pipefail
BASE=$1 CAND=$2 EV=$3 REPS=${4:-3} WORKLOADS=${5:-"onebox multi-instance"}
RUN=$(realpath "$(dirname "$0")/run-cells.sh")
test -x "${FN182_BASE_BINARY:?prebuild the baseline qualification test binary}"
test -x "${FN182_CAND_BINARY:?prebuild the candidate qualification test binary}"
test ! -e "$EV"
mkdir -p "$EV"
{ date -u +%FT%TZ; uname -a; cat /proc/sys/kernel/random/boot_id; nproc; free -b; grep -m4 -E "CPU implementer|CPU architecture|CPU part|Features" /proc/cpuinfo;
  for w in "$BASE" "$CAND"; do echo "$w $(git -C "$w" rev-parse HEAD)"; git -C "$w" status --porcelain --untracked-files=no; done; } > "$EV/host.txt"
for rep in $(seq 1 "$REPS"); do
  if [ $((rep % 2)) -eq 1 ]; then order="base cand"; else order="cand base"; fi
  for side in $order; do
    wt=$BASE binary=$FN182_BASE_BINARY
    if [ "$side" = cand ]; then wt=$CAND binary=$FN182_CAND_BINARY; fi
    for w in $WORKLOADS; do
      root="$EV/$side-$w-rep$rep"
      echo "$(date -u +%FT%TZ) START $side $w $rep load=$(cut -d' ' -f1-3 /proc/loadavg)" >> "$EV/stages.log"
      { date -u +%FT%TZ; cat /proc/loadavg; free -b; ps -eo pid,ppid,comm,%cpu,rss --sort=-%cpu | head -20; } > "$EV/$side-$w-rep$rep-host.txt" || true
      FN182_QUALIFICATION_BINARY="$binary" "$RUN" "$wt" "$root" "$w:success"
      echo "$(date -u +%FT%TZ) EXIT $side $w $rep 0" >> "$EV/stages.log"
    done
  done
done
python3 -I "$(dirname "$0")/summarize.py" "$EV" > "$EV/summary.md"
