#!/usr/bin/env bash
# fn-182 R4 matched baseline/candidate measurement (QUIET HOST ONLY; wait for GO MEASURE).
# Usage: measure-r4.sh <baseline-worktree> <candidate-worktree> <evidence-root> [reps=5] [workloads="onebox multi-instance"]
# Each rep runs the existing qualification success cell in a fresh evidence root (cold Go
# preparation, cold engine compile on execution 1, warm cache hits afterwards), alternating
# ABBA order. Both helpers must be built in their own worktree beforehand (make -C tools/gomad wasm-helper).
set -uo pipefail
BASE=$1 CAND=$2 EV=$3 REPS=${4:-5} WORKLOADS=${5:-"onebox multi-instance"}
RUN=$(dirname "$0")/run-cells.sh
mkdir -p "$EV"
{ date -u +%FT%TZ; uname -a; cat /proc/sys/kernel/random/boot_id; nproc; free -b; grep -m4 -E "CPU implementer|CPU architecture|CPU part|Features" /proc/cpuinfo;
  for w in "$BASE" "$CAND"; do echo "$w $(git -C "$w" rev-parse HEAD)"; git -C "$w" status --porcelain --untracked-files=no; done; } > "$EV/host.txt"
quiet() { for i in $(seq 1 180); do l=$(cut -d' ' -f1 /proc/loadavg); m=$(awk '/MemAvailable/{print $2}' /proc/meminfo);
  awk -v l="$l" -v m="$m" 'BEGIN{exit !(l<1.5 && m>=8*1024*1024)}' && return 0; sleep 10; done; echo "host not quiet" >&2; return 1; }
for rep in $(seq 1 "$REPS"); do
  if [ $((rep % 2)) -eq 1 ]; then order="base cand"; else order="cand base"; fi
  for side in $order; do
    wt=$BASE; [ "$side" = cand ] && wt=$CAND
    for w in $WORKLOADS; do
      quiet || exit 3
      root="$EV/$side-$w-rep$rep"
      echo "$(date -u +%FT%TZ) START $side $w $rep load=$(cut -d' ' -f1-3 /proc/loadavg)" >> "$EV/stages.log"
      "$RUN" "$wt" "$root" "$w:success"
      echo "$(date -u +%FT%TZ) EXIT $side $w $rep $(grep -c 'EXIT .* 0$' "$root/stages.log")/1" >> "$EV/stages.log"
    done
  done
done
python3 -I "$(dirname "$0")/summarize.py" "$EV" > "$EV/summary.md"
