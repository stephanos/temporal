#!/usr/bin/env bash
# Reruns the fn-193.1 plain-composition probe: compile it with the framework's DSL and realize
# sources (not their tests) under the framework's strict flags, run its suite, and print the report
# (members, syncs, a sync's step, claim verdicts, the witness, the refuted control, the refusals).
# Run from anywhere; logs land in logs/.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../../.." && pwd)" # the repository root (umpire/temporal), where mise.toml pins scala-cli
cd "$root"
logs="$here/logs"
mkdir -p "$logs"
p="model/design/plain-composition-prototype"
scala() { mise exec -- scala-cli "$@" --server=false; }

# The framework's main sources and its project file (Scala, JVM, flags, IR jar, munit); no framework
# test is compiled or run here. The probe's DSL is main scope; its fixtures, suite and report are
# test scope (see the README).
inputs=(model/framework/dsl/project.scala)
for f in model/framework/dsl/*.scala model/framework/realize/*.scala; do
  case "$f" in *.test.scala | */project.scala) ;; *) inputs+=("$f") ;; esac
done
inputs+=("$p")

scala compile --test "${inputs[@]}" >"$logs/compile.log" 2>&1
echo "compile: exit $?" | tee "$logs/compile.status"

scala test "${inputs[@]}" >"$logs/test.log" 2>&1
status=$?
passed=$(grep -c ' + ' "$logs/test.log")
failed=$(grep -c '==> X' "$logs/test.log")
echo "test: exit $status, $passed passed, $failed failed" | tee "$logs/test.status"

scala run --test "${inputs[@]}" --main-class framework.dsl.compose.report >"$logs/report.log" 2>&1
report=$?
echo "report: exit $report" | tee "$logs/report.status"
grep -v -e JAVA_TOOL -e WARNING -e 'hint' "$logs/report.log"
[ $status -eq 0 ] && [ $report -eq 0 ]
