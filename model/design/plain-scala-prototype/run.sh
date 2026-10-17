#!/usr/bin/env bash
# Reruns the fn-190.1 plain-Scala probe: compile under the strict flags, the native suites, the
# report (schema, arms, verdicts, progress, reload cost), and the non-exhaustive-match refusal.
# Run from anywhere; logs land in logs/.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../../.." && pwd)" # the repository root (umpire/temporal), where mise.toml pins scala-cli
cd "$root"
logs="$here/logs"
mkdir -p "$logs"
p="model/design/plain-scala-prototype"
scala() { mise exec -- scala-cli "$@" --server=false; }

scala compile "$p" >"$logs/compile.log" 2>&1
echo "compile: exit $?" | tee "$logs/compile.status"

scala test "$p" >"$logs/test.log" 2>&1
status=$?
passed=$(grep -c '^.*+ ' "$logs/test.log")
failed=$(grep -c '==> X' "$logs/test.log")
echo "test: exit $status, $passed passed, $failed failed" | tee "$logs/test.status"

scala run "$p" --main-class proto.report >"$logs/report.log" 2>&1
echo "report: exit $?" | tee "$logs/report.status"
grep -v -e JAVA_TOOL -e WARNING "$logs/report.log"

# A phase match missing a case must not build: compile the DSL with the negative probe alone.
neg="$(mktemp -d)"
cp "$p/negative/NonExhaustive.scala.txt" "$neg/NonExhaustive.scala"
scala compile "$p/project.scala" "$p/dsl/Dsl.scala" "$p/dsl/Macros.scala" "$p/dsl/Runtime.scala" \
  "$neg/NonExhaustive.scala" >"$logs/non-exhaustive.log" 2>&1
neg_status=$?
rm -rf "$neg"
if [ $neg_status -ne 0 ] && grep -q "may not be exhaustive" "$logs/non-exhaustive.log"; then
  echo "non-exhaustive match: refused (exit $neg_status)" | tee "$logs/non-exhaustive.status"
else
  echo "non-exhaustive match: NOT refused (exit $neg_status)" | tee "$logs/non-exhaustive.status"
  exit 1
fi
[ $status -eq 0 ]
