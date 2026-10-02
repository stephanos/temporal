#!/bin/bash
# Runs a ./tests selection under Gomad and keeps each seed's test output (gzip) for analysis.
#
# usage: D20_OUT=<scratch dir> fn105-d20-run.sh <name> <seeds> <strict|forward> <test.run pattern> [source-dir]
#
# Output goes to $D20_OUT, which must be a directory inside the session scratchpad; one
# subdirectory per <name>. Campaign artifacts are removed through the scratchpad's safe-rm.sh
# (it refuses any path outside the scratchpad) after stdout/stderr are extracted. [source-dir]
# defaults to the repository that holds this script; a copy of the source tree can be passed to
# run a variant without touching the working tree.
set -euo pipefail
HERE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
SOURCE_ROOT=${D20_REPO:-$(cd -- "$HERE/../../.." && pwd -P)}
SCRATCHPAD=/private/tmp/claude-501/-Users-stephan-Workspace-temporal-gomad/f662b53b-adef-414e-85f5-25221ab92e04/scratchpad
SAFE_RM=${D20_SAFE_RM:-$SCRATCHPAD/safe-rm.sh}
OUT=${D20_OUT:?set D20_OUT to a directory inside the scratchpad}
case "$OUT" in "$SCRATCHPAD"/?*) ;; *) echo "D20_OUT must be inside $SCRATCHPAD" >&2; exit 2;; esac
[ "$#" -ge 4 ] || { echo "usage: $0 <name> <seeds> <strict|forward> <pattern> [source-dir]" >&2; exit 2; }
name=$1; seeds=$2; tick=$3; pat=$4; dir=${5:-$SOURCE_ROOT}
case "$name" in *[!A-Za-z0-9_.-]*|"") echo "bad name: $name" >&2; exit 2;; esac
export PATH="$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH"
art="$OUT/$name-artifacts"
cd -- "$dir"
if [ -e "$art" ]; then "$SAFE_RM" "$art"; fi
mkdir -p "$OUT/out/$name"
start=$(date +%s)
rc=0
env -u GOMADSEED -u GOMAD3_CHILD_SEED "$SOURCE_ROOT/tools/gomad3/.bin/gomad" explore --seeds "$seeds" --parallel 2 --on-failure=all \
  --clock-tick="$tick" --build-tag=disable_grpc_modules --build-tag=gomad --build-tag=test_dep --capability-mode=closure \
  --io-ro-mount ./schema=/go.temporal.io/server/schema --keep-successes=all --success-limit=64 --success-bytes=8GiB \
  --execution-timeout=8m --overall-timeout=9m --artifacts "$art" \
  go-test ./tests -- "-test.run=$pat" -test.parallel=8 -test.v > "$OUT/out/$name/gomad.log" 2>&1 || rc=$?
echo "name=$name seeds=$seeds tick=$tick pattern=$pat exit=$rc wall=$(( $(date +%s) - start ))s output=$OUT/out/$name"
{ grep "classification=" "$OUT/out/$name/gomad.log" || true; } | sed 's/ artifact=.*//'
for d in "$art"/v1/campaign-*/failures/* "$art"/v1/campaign-*/successes/*; do
  [ -d "$d" ] || continue
  seed=$( { grep -o '"seed":"[0-9]*"' "$d/manifest.json" || true; } | head -1 | tr -cd '0-9')
  kind=$(basename "$(dirname "$d")")
  gzip -c "$d/stdout" > "$OUT/out/$name/seed$seed-stdout.gz"
  gzip -c "$d/stderr" > "$OUT/out/$name/seed$seed-stderr.gz"
  res=$( { grep -E '^\s*--- (PASS|FAIL).*TestWorkflowTaskHeartbeatingWithEmptyResult' "$d/stdout" || true; } | sed 's/^ *//' | tr '\n' ' ')
  # Only the D20-instrumented test (fn105-d20-variant-i-instrumentation.diff.txt) logs rejections.
  hb=$( { grep -c 'D20 iter=.*workflow task heartbeat timeout' "$d/stdout" || true; } )
  echo "seed=$seed $kind :: $res :: instrumented-rejections=$hb"
done | sort -t= -k2 -n
if [ -e "$art" ]; then "$SAFE_RM" "$art"; fi
