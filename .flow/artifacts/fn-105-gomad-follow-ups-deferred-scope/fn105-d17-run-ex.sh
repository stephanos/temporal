#!/bin/bash
# usage: run-ex.sh NAME SEEDS TICK CLUSTERS(0=unset) PATTERN WORKDIR [TEST_TIMEOUT]
# Runs gomad explore with -test.v once. Never deletes anything: NAME must be new.
set -euo pipefail
D=/private/tmp/claude-501/-Users-stephan-Workspace-temporal-gomad/f662b53b-adef-414e-85f5-25221ab92e04/scratchpad/d17
GOMAD=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.bin/gomad
[ "$#" -ge 6 ] || { echo "usage" >&2; exit 2; }
name="$1"; seeds="$2"; tick="$3"; clusters="$4"; pat="$5"; workdir="$6"; ttimeout="${7:-}"
case "$name" in *[!A-Za-z0-9._-]*|"") echo "bad name" >&2; exit 2;; esac
[ -d "$workdir/tests" ] || { echo "no workdir" >&2; exit 2; }
[ ! -e "$D/art/$name" ] || { echo "artifacts dir exists" >&2; exit 2; }
[ ! -e "$D/out/$name" ] || { echo "output dir exists" >&2; exit 2; }
export PATH="$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH"
args=(explore --seeds "$seeds" --parallel 2 --on-failure=all "--clock-tick=$tick" --build-tag=disable_grpc_modules --build-tag=gomad --build-tag=test_dep --capability-mode=closure --io-ro-mount ./schema=/go.temporal.io/server/schema --keep-successes=all --success-limit=64 --success-bytes=8GiB --execution-timeout=2m --overall-timeout=15m)
if [ "$clusters" != 0 ]; then args+=("--env=TEMPORAL_TEST_DEDICATED_CLUSTERS=$clusters"); fi
args+=(--artifacts "$D/art/$name" go-test ./tests -- "-test.run=$pat" -test.parallel=8 -test.v)
if [ -n "$ttimeout" ]; then args+=("-test.timeout=$ttimeout"); fi
mkdir -p "$D/out/$name"
start=$(date +%s)
cd "$workdir"
rc=0
env -u GOMADSEED -u GOMAD3_CHILD_SEED "$GOMAD" "${args[@]}" > "$D/out/$name/gomad.log" 2>&1 || rc=$?
echo "name=$name seeds=$seeds tick=$tick clusters=$clusters pattern=$pat test_timeout=$ttimeout exit=$rc wall=$(( $(date +%s) - start ))s load=$(uptime | sed 's/.*averages: //')"
grep "classification=" "$D/out/$name/gomad.log" | sed 's/ artifact=.*//' | tail -1
for d in "$D/art/$name"/v1/campaign-*/failures/* "$D/art/$name"/v1/campaign-*/successes/*; do
  [ -d "$d" ] || continue
  seed=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['seed'])" "$d/manifest.json")
  kind=$(basename "$(dirname "$d")")
  cp "$d/stdout" "$D/out/$name/seed$seed-stdout.txt"; cp "$d/stderr" "$D/out/$name/seed$seed-stderr.txt"; cp "$d/manifest.json" "$D/out/$name/seed$seed-manifest.json"
  echo "seed=$seed $kind stdout=$(wc -c < "$d/stdout") stderr=$(wc -c < "$d/stderr") :: $(grep -E '^\s*--- (PASS|FAIL|SKIP)' "$d/stdout" | tr -s ' ' | tr '\n' ';' || true)"
done
