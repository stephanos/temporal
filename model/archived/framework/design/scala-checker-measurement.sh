#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
task_out="${UMPIRE_MEASUREMENT_OUT:-$PWD/.flow/tmp/fn1963/measure}"
mkdir -p "$task_out"
task_sources=(model/framework/checker model/framework/samples/Bank.scala model/framework/samples/Voting.scala model/framework/samples/TwoPhaseCommit.scala model/framework/samples/Kettle.scala model/framework/samples/ReliableBroadcast.scala model/framework/samples/TwoLayeredCache.scala model/design/scala-checker-measurement.scala)
if [ "${1:-}" = build ]; then
  if [ -e "$task_out/compile.log" ] || [ -e "$task_out/classpath.log" ]; then
    printf 'Use a fresh UMPIRE_MEASUREMENT_OUT to preserve prior build evidence.\n' >&2
    exit 2
  fi
  rg --files model/framework/dsl model/framework/realize model/framework/checker "${task_sources[@]:1}" model/OutputPaths.scala model/build/ir-scalapb.jar | sort -u | xargs sha256sum > "$task_out/source.sha256"
  timeout 600 env -u BASH_ENV UMPIRE_FRAMEWORK_BUILD="$task_out/build" mise exec -- scala-cli compile --server=false --workspace "$task_out/workspace" --print-class-path --compilation-output "$task_out/measurement.jar" "${task_sources[@]}" > "$task_out/classpath.log" 2> "$task_out/compile.log"
  exit 0
fi
test -s "$task_out/measurement.jar"
task_cp=$(tail -1 "$task_out/classpath.log")
task_cp="$task_out/measurement.jar:${task_cp#*:}"
task_gc="$task_out/gc-${1}-${2:-}.log"
if [ -e "$task_gc" ]; then
  printf 'Refusing to overwrite GC evidence: %s\n' "$task_gc" >&2
  exit 2
fi
exec timeout 600 env -u BASH_ENV mise exec -- java -Xms256m -Xmx2g -XX:+UseSerialGC "-Xlog:gc:file=$task_gc:uptime,level,tags" -cp "$task_cp" framework.checker.ScalaCheckerMeasurement "$@"
