#!/usr/bin/env bash
# Incremental-compilation staleness probe: build with the private Bloop server, print the reified
# temperature body, edit only the helper's body in its own file, rebuild incrementally, print again.
set -euo pipefail
P=/tmp/umpire-macro-spike/proto; S=/tmp/umpire-macro-spike/scripts/sc; D=/tmp/umpire-macro-spike/stale
B="--bloop-daemon-dir /tmp/umpire-macro-spike/bloop-daemon"
mode=${1:-server}
if [ "$mode" = server ]; then srv="$B"; else srv="--server=false"; fi
H=$D/samples/KettleHelpers.scala
sed -i '' 's/state.reading + Kettle.floor + 1$/state.reading + Kettle.floor/' $H
rm -rf $D/samples/.scala-build $P/model/.scala-build
build() { cd /tmp && $S run $srv --suppress-outdated-dependency-warning $P/model/project.scala $P/model/framework/dsl $P/model/framework/realize $D/samples $P/export -M framework.reify.exportKettle -- $D/out.json >/dev/null 2>$D/build.log; python3 -I -c '
import json
for f in json.load(open("/tmp/umpire-macro-spike/stale/out.json"))["functions"]:
    if f["name"] == "samples.KettleHelpers$.temperature":
        print(json.dumps(f["body"]["binary"]["right"]))'; }
echo "== before edit ($mode)"; build
sed -i '' 's/state.reading + Kettle.floor$/state.reading + Kettle.floor + 1/' $H
echo "== after editing only KettleHelpers.temperature's body ($mode)"; build
grep -n "temperature" $H
