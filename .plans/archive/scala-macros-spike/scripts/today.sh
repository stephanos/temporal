#!/usr/bin/env bash
# Today's pipeline for one sample (mirrors framework/check/Samples.scala and the kettle-check run.sh)
# on the /tmp copy of the repo: package the Models' TASTy, print the classpath, lift in its own JVM.
# usage: today.sh <outdir> [scala-cli server options...]   (default --server=false, as the Makefile)
set -euo pipefail
repo=/tmp/umpire-macro-spike/repo
out=$1; shift; mkdir -p "$out"
srv=("$@"); [ ${#srv[@]} -eq 0 ] && srv=(--server=false)
S=/tmp/umpire-macro-spike/scripts/sc
inputs="$repo/model/project.scala $repo/model/framework/dsl $repo/model/framework/realize $repo/model/framework/samples"
cd $repo   # the lifter reads source lines relative to the working directory
now() { python3 -c 'import time;print(time.time())'; }
t0=$(now)
$S --power package "${srv[@]}" --suppress-outdated-dependency-warning --library $inputs -f -o $out/samples.jar >/dev/null 2>$out/package.log
t1=$(now)
$S compile "${srv[@]}" --suppress-outdated-dependency-warning --print-class-path $inputs > $out/samples.classpath 2>$out/cp.log
t2=$(now)
$S run "${srv[@]}" --suppress-outdated-dependency-warning $repo/model/framework/export --main-class framework.export.lift -- \
    "$out/samples.jar=model/" $out/samples.classpath $out/Kettle.json 'samples.Kettle$.queries' 2>$out/lift.log
t3=$(now)
python3 -c "import sys,json;a=list(map(float,sys.argv[1:]));print(json.dumps({'package':a[1]-a[0],'classpath':a[2]-a[1],'lift':a[3]-a[2],'total':a[3]-a[0]}))" $t0 $t1 $t2 $t3
