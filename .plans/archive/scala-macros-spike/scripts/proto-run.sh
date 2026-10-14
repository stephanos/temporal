#!/usr/bin/env bash
# Rebuilds the reifier jar when asked, then compiles the prototype Model and runs its exporter.
# usage: proto-run.sh [--rebuild-reify] [server-flag] [extra sample dir]
set -euo pipefail
P=/tmp/umpire-macro-spike/proto; S=/tmp/umpire-macro-spike/scripts/sc; O=/tmp/umpire-macro-spike/out
if [ "${1:-}" = "--rebuild-reify" ]; then shift; (cd $P/reify && $S --power package --server=false --library . -f -o $P/reify.jar 2>&1 | grep -v "^\[warn\]\|Using directives\|It is recommended\|^- " | tail -3); fi
srv=${1:---server=false}
cd /tmp
$S run $srv --suppress-outdated-dependency-warning $P/model/project.scala $P/model/framework/dsl $P/model/framework/realize $P/model/framework/samples $P/export -M framework.reify.exportKettle -- $O/reified.json $O/today0/Kettle.json $O/lifted-subset.json 2>&1 | grep -v "WARNING\|Using directives\|It is recommended\|^- \|^\[warn\]\|more than one project.scala\|^$"
