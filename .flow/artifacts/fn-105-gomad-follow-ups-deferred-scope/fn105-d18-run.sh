#!/bin/bash
# Runs TestWorkerCommandsTaskSuite (or one of its tests) under Gomad and prints, per seed, which
# equal-deadline timer task ran first and whether a WorkerCommands task was created.
#
# usage: fn105-d18-run.sh <name> <seeds> <strict|forward> <test.run pattern> [source-dir]
#
# Output goes to $D18_OUT (default: a fresh temporary directory), one subdirectory per <name>,
# in the layout fn105-d18-summarize.py reads. Campaign artifacts are deleted after the timelines
# are extracted. [source-dir] defaults to the repository that holds this script; a copy of the
# source tree can be passed to run a variant without touching the working tree.
set -u
HERE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
REPO=$(cd -- "$HERE/../../.." && pwd -P)
OUT=${D18_OUT:-$(mktemp -d "${TMPDIR:-/tmp}/fn105-d18.XXXXXX")}
name=$1; seeds=$2; tick=$3; pat=$4; dir=${5:-$REPO}
export PATH=$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH
cd "$dir" || exit 1
rm -rf "$OUT/$name-artifacts"; mkdir -p "$OUT/out/$name"
start=$(date +%s)
env -u GOMADSEED -u GOMAD3_CHILD_SEED "$REPO/tools/gomad3/.bin/gomad" explore --seeds "$seeds" --parallel 2 --on-failure=all \
  --clock-tick="$tick" --build-tag=disable_grpc_modules --build-tag=gomad --build-tag=test_dep --capability-mode=closure \
  --io-ro-mount ./schema=/go.temporal.io/server/schema --keep-successes=all --success-limit=64 --success-bytes=8GiB \
  --execution-timeout=8m --overall-timeout=9m --artifacts "$OUT/$name-artifacts" \
  go-test ./tests -- "-test.run=$pat" -test.parallel=8 -test.v > "$OUT/out/$name/gomad.log" 2>&1
rc=$?
echo "name=$name seeds=$seeds tick=$tick pattern=$pat exit=$rc wall=$(( $(date +%s) - start ))s output=$OUT/out/$name"
grep "classification=" "$OUT/out/$name/gomad.log" | sed 's/ artifact=.*//'
for d in "$OUT/$name-artifacts"/v1/campaign-*/failures/* "$OUT/$name-artifacts"/v1/campaign-*/successes/*; do
  [ -d "$d" ] || continue
  seed=$(grep -o '"seed":"[0-9]*"' "$d/manifest.json" | head -1 | grep -o '[0-9]*')
  kind=$(basename "$(dirname "$d")")
  python3 "$HERE/fn105-d18-timeline.py" "$d/stderr" "$d/stdout" > "$OUT/out/$name/seed$seed-timeline.txt"
  grep -E '^(---|    ---|FAIL|PASS|ok)' "$d/stdout" > "$OUT/out/$name/seed$seed-results.txt"
  grep -A40 'attempt errors:' "$d/stdout" | grep -v shared_cluster_t > "$OUT/out/$name/seed$seed-await-report.txt"
  first=$(grep -E 'task=(WorkflowRunTimeoutTask|ActivityTimeoutTask)' "$OUT/out/$name/seed$seed-timeline.txt" | head -1 | grep -oE 'task=[A-Za-z]+')
  cmd=$(grep -c 'queue-task-type=WorkerCommands' "$OUT/out/$name/seed$seed-timeline.txt")
  res=$(grep 'TestDispatchCancelOnWorkflowTimeout' "$OUT/out/$name/seed$seed-results.txt" | sed 's/^ *//')
  echo "seed=$seed $kind first_timer_$first worker_commands_tasks=$cmd :: $res"
done | sort -t= -k2 -n
rm -rf "$OUT/$name-artifacts"
