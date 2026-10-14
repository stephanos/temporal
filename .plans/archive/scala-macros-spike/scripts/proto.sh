#!/usr/bin/env bash
# The prototype's loop: one scala-cli run compiles the Model (macros reify the rules while it
# compiles) and runs the exporter, which writes the IR and compares it with today's lift.
# usage: proto.sh <outdir> [scala-cli server options...]
set -euo pipefail
P=/tmp/umpire-macro-spike/proto; S=/tmp/umpire-macro-spike/scripts/sc
out=$1; shift; mkdir -p "$out"
srv=("$@"); [ ${#srv[@]} -eq 0 ] && srv=(--server=false)
cd /tmp
now() { python3 -c 'import time;print(time.time())'; }
t0=$(now)
REIFY_LOG=$out/macro-times.log $S run "${srv[@]}" --suppress-outdated-dependency-warning $P/model/project.scala $P/model/framework/dsl $P/model/framework/realize $P/model/framework/samples $P/export -M framework.reify.exportKettle -- $out/reified.json /tmp/umpire-macro-spike/out/today0/Kettle.json $out/lifted-subset.json >$out/run.log 2>&1
t1=$(now)
python3 -c "import sys,json;a=list(map(float,sys.argv[1:]));print(json.dumps({'total':a[1]-a[0]}))" $t0 $t1
