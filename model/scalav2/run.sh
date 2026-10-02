#!/usr/bin/env bash
# The model/scalav2 gate. Scala is only the authoring front end: the Models in scala/ are compiled
# and tested, the lifter reads their typed trees and emits the IR, and Go interprets the IR and checks
# every table, identity and fingerprint against the Lean Model. Exits non-zero on the first failure.
#
#   model/scalav2/run.sh           lift, require ir/nexus-caller.json, ir/nexus-close.json and ir/activity*.json to be current, test
#   model/scalav2/run.sh --update  lift and rewrite ir/nexus-caller.json, ir/nexus-close.json and ir/activity*.json
#
# The lifter's fixtures under lifter/testdata are built and lifted too: the Models it must lift,
# compared with the IR in lifter/testdata/lifts/expected, which --update rewrites as well, and the
# declarations the build or the lifter must refuse, at their lines.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
cd "$root"

update=false
for arg in "$@"; do
  case "$arg" in
    --update) update=true ;;
    *) echo "usage: run.sh [--update]" >&2; exit 2 ;;
  esac
done

scala_cli() { mise exec -- scala-cli "$@" --suppress-outdated-dependency-warning; }
# scala-cli exits 0 when -Werror turns a warning into an error, such as a match that misses a case;
# scala.sh fails on any error it prints. Its output is shown only when the build fails.
scala_build() {
  local out
  out="$("$here/scala.sh" "$@")" || { echo "$out" >&2; return 1; }
}
# The lifter's own arguments follow `--`, so scala-cli's flags go before it: an argument after it
# is a root, and the lifter refuses a root that names nothing.
lift() {
  local diagnostics lift_rc=0
  diagnostics="$(mktemp "$here/gen/lift-stderr.XXXXXX")" || return $?
  mise exec -- scala-cli run --suppress-outdated-dependency-warning "$here/lifter" -- "$@" \
    2> "$diagnostics" || lift_rc=$?
  sed '/^WARNING/d' "$diagnostics" >&2 || return $?
  rm "$diagnostics" || return $?
  return "$lift_rc"
}
# A fixture's sources are stored as <file>.scala.fixture, which no build, formatter or linter of the
# tree reads, since some of them must not compile. materialize <fixture> copies them into
# gen/fixtures/<fixture> as the .scala files its build reads, with its project's jar path resolved,
# and prints that directory. The lifter maps the copies' positions back to the stored files, and the
# build's diagnostics are mapped back below.
trap 'rm -rf "$here/gen/fixtures"' EXIT
materialize() {
  local from="$here/lifter/testdata/$1" to="$here/gen/fixtures/$1" f
  rm -rf "$to" && mkdir -p "$to"
  for f in "$from"/*.scala.fixture; do
    sed "s#\.\./\.\./\.\./gen/model-scala\.jar#$here/gen/model-scala.jar#" "$f" > "$to/$(basename "$f" .fixture)"
  done
  echo "$to"
}
# The lifter's positions of a materialized fixture: its stored files.
stored() { echo "model/scalav2/lifter/testdata/$1/%s.fixture"; }
# refuses <fixture> <file:line>...: the fixture's build fails, with an error at each line of the
# stored file.
refuses() {
  local fixture="$1" refused line
  shift
  refused="$(scala_build compile "$(materialize "$fixture")" 2>&1 && echo "built" || true)"
  refused="$(sed -e 's/\x1b\[[0-9;]*m//g' \
    -e "s#\./model/scalav2/gen/fixtures/$fixture/\([^:]*\)\.scala:#./model/scalav2/lifter/testdata/$fixture/\1.scala.fixture:#" \
    <<<"$refused")"
  for line in "$@"; do
    grep -qF "[error] ./model/scalav2/lifter/testdata/$fixture/$line" <<<"$refused" \
      || { echo "run.sh: expected lifter/testdata/$fixture to fail at $line, got:" >&2; echo "$refused" >&2; exit 1; }
  done
  grep -F "[error] ./model/scalav2/lifter/testdata/$fixture/" <<<"$refused"
}
# The functional Queries and the realization that runs them are roots beside the machines: Go lowers
# each Query's witness through the realization into a Testpilot Case (goir/testpilot).
roots=('temporal.nexuscaller.Model$package$.nexusProduct' 'temporal.nexuscaller.Model$package$.nexusProtocol'
  'temporal.nexuscaller.Model$package$.handlerWorker' 'temporal.worker.Worker$package$.polling'
  'temporal.nexuscaller.Claims$package$.functionalQueries' 'temporal.nexuscaller.NexusRealization$.asyncNexus')
control_roots=('temporal.nexuscaller.Control$.forgedCompletion' 'temporal.nexuscaller.NexusRealization$.forgedCompletion')

# The standalone activity Model as model/go/standaloneactivity has it: ir/activity.json. Its
# cross-entity Query, stoppedWorkerStartsNothing, carries the composition and its claim.
activity='temporal.standaloneactivity.Model$package$.'
activity_claims='temporal.standaloneactivity.Claims$package$.'
activity_roots=("${activity}standaloneActivity" "${activity}activityProduct" "${activity_claims}functionalQueries"
  "${activity_claims}terminalHolds" "${activity_claims}pauseHolds" "${activity_claims}cancelRequest"
  "${activity_claims}stoppedWorkerStartsNothing" 'temporal.standaloneactivity.ActivityRealization$.standalone')
# Its system contract, the admission designs and the dispatch queue's providers: ir/activity-system.json.
# A composition no Query runs over is a root of its own.
activity_system_roots=(currentQueries staleQueries competingTimers matchingQueueQueries forgetfulQueueQueries
  volatileQueueQueries lossyMatchingQueueQueries storageLossQuery currentOverQueueQueries staleOverQueueQueries
  currentOverMatchingQueries staleOverMatchingQueries currentOverLossyMatchingQueries currentOverForgetful
  currentOverVolatile)
# The held race a server is run through, and the realization that runs it: ir/activity-race.json. It
# is a Model of its own, so the system contract's Queries are the ones its checkers were given.
activity_race_roots=('temporal.standaloneactivity.System$package$.heldStaleDelivery'
  'temporal.standaloneactivity.ActivityRealization$.heldDelivery'
  'temporal.standaloneactivity.System$package$.lostAdmissionResponseQuery'
  'temporal.standaloneactivity.ActivityRealization$.lostAdmissionResponse')

# The Nexus caller close and reset designs: ir/nexus-close.json. Each design's Queries are a root, and
# so is each progress claim.
nexus_close_roots=(rejectAfterCloseQueries ackByOriginalQueries retainAndRouteQueries forgetsCancelOnResetQueries
  truncatesOnResetQueries retainAndRouteBoundedRetryQueries rejectAfterCloseWithDeadlineQueries
  ackByOriginalWithDeadlineQueries retainAndRouteWithDeadlineQueries rejectAfterCloseProgress ackByOriginalProgress
  retainAndRouteProgress retainAndRouteBoundedRetryProgress rejectAfterCloseWithDeadlineProgress
  ackByOriginalWithDeadlineProgress retainAndRouteWithDeadlineProgress retainedReachesOwner
  retainedWaitsWithoutRecovery retainedReachesOwnerBoundedRetry)

schema="$root/proto/internal/temporal/server/api/modelir/v1/ir.proto"
echo "== generate the IR's Java classes when their inputs changed"
"$here/gen.sh" --if-stale
[[ api/modelir/v1/ir.pb.go -nt "$schema" ]] || { echo "run.sh: api/modelir/v1 is older than the IR schema; run make protoc" >&2; exit 1; }

# The framework must build without the Temporal Models, so nothing in umpire/ reaches into them.
scala="model/scalav2/scala"
echo "== compile the framework alone"
scala_build compile "$scala/project.scala" "$scala/umpire"

echo "== compile and test the framework and the Temporal Models"
scala_build test "$scala/project.scala" "$scala/umpire" "$scala/temporal"

echo "== package the Models' TASTy"
scala_build --power package --library "$scala/project.scala" "$scala/umpire" "$scala/temporal" \
  -f -o "$here/gen/model-scala.jar"
scala_cli compile --print-class-path "$scala/project.scala" "$scala/umpire" "$scala/temporal" \
  > "$here/gen/model-scala.classpath"

echo "== the build refuses a warning -Werror makes an error, and crossed types, at their lines"
refuses werror Evidence.scala.fixture:29:16
refuses crossed Crossed.scala.fixture:35:14 Crossed.scala.fixture:45:28

echo "== lift the Nexus caller Model"
lifted="$(mktemp)"
lift "$here/gen/model-scala.jar=$scala/" "$here/gen/model-scala.classpath" "$lifted" "${roots[@]}"
if $update; then
  cp "$lifted" "$here/ir/nexus-caller.json"
elif ! diff -q "$here/ir/nexus-caller.json" "$lifted" >/dev/null; then
  # diff exits 1 on a difference, which under pipefail would stop the script before it says why.
  diff "$here/ir/nexus-caller.json" "$lifted" | head -20 || true
  echo "run.sh: ir/nexus-caller.json is stale; rerun with --update" >&2
  exit 1
fi
rm -f "$lifted"

echo "== the lifter refuses a construct outside the subset, at its line"
fixture="$(materialize unsupported)"
scala_build --power package --library "$fixture" -f -o "$here/gen/unsupported.jar"
refused="$(lift "$here/gen/unsupported.jar=$(stored unsupported)" "$here/gen/model-scala.classpath" \
  /dev/null 'temporal.fixture.Unsupported$package$.unsupported' 2>&1 | grep '^lift:' || true)"
expected='lift: model/scalav2/lifter/testdata/unsupported/Unsupported.scala.fixture:18: `var out` has no IR form'
[[ "$refused" == "$expected"* ]] || { echo "run.sh: expected '$expected ...', got '$refused'" >&2; exit 1; }
echo "$refused"

echo "== lift the fixtures, the activity Models and the Nexus close designs, and compare them with lifter/testdata/lifts/expected and ir"
scala_build --power package --library "$(materialize lifts)" -f -o "$here/gen/lifts.jar"
jars="$here/gen/lifts.jar=$(stored lifts),$here/gen/model-scala.jar=$scala/"
rm -rf "$here/gen/lifts" && mkdir -p "$here/gen/lifts"
# Each lift is its own JVM, so they run side by side: <name> <jars> <root>...
lift_into() {
  local name="$1" from="$2"
  shift 2
  local status=0
  lift "$from" "$here/gen/model-scala.classpath" "$here/gen/lifts/$name.json" "$@" > "$here/gen/lifts/$name.log" 2>&1 \
    || status=$?
  echo "$status" > "$here/gen/lifts/$name.status"
}
lift_into presence "$jars" 'fixture.presence.Presence$package$.presence' &
lift_into channels "$jars" 'fixture.channels.Channels$package$.relay' \
  'fixture.channels.Channels$package$.tallying' &
lift_into declarations "$jars" 'fixture.declarations.Declarations$package$.queries' \
  'fixture.declarations.Declarations$package$.durableEventually' &
lift_into admission "$jars" 'fixture.specimens.admission.Admission$package$.currentQueries' \
  'fixture.specimens.admission.Admission$package$.staleQueries' &
lift_into closereset "$jars" 'fixture.specimens.closereset.CloseReset$package$.rejectAfterCloseQueries' \
  'fixture.specimens.closereset.CloseReset$package$.ackByOriginalQueries' \
  'fixture.specimens.closereset.CloseReset$package$.retainAndRouteQueries' &
lift_into realizations "$jars" 'fixture.realizations.Realizations$package$.learnedRun' \
  'temporal.nexuscaller.Claims$package$.syncCompletion' 'fixture.realizations.Realizations$package$.pauseRace' \
  'fixture.realizations.Realizations$package$.pauseRaceQuery' \
  'fixture.realizations.Realizations$package$.doorRealization' 'fixture.realizations.Realizations$package$.doorOpens' \
  'fixture.realizations.Realizations$package$.errandRealization' 'fixture.realizations.Realizations$package$.errandRetry' \
  'fixture.realizations.Realizations$package$.errandWithdrawn' \
  'fixture.realizations.Realizations$package$.tallyRealization' 'fixture.realizations.Realizations$package$.tallyOpens' &
rejected=(unbounded waiting doubled listening counter crossedRead negative watched unrefined misplaced noSuchRoot
  unrefinedOutcomes counting batching shuffling guessing)
lift_into rejects "$jars" "${rejected[@]/#/fixture.rejects.Rejects\$package\$.}" &
lift_into activity "$here/gen/model-scala.jar=$scala/" "${activity_roots[@]}" &
lift_into activity-system "$here/gen/model-scala.jar=$scala/" \
  "${activity_system_roots[@]/#/temporal.standaloneactivity.System\$package\$.}" &
lift_into activity-race "$here/gen/model-scala.jar=$scala/" "${activity_race_roots[@]}" &
lift_into nexus-control "$here/gen/model-scala.jar=$scala/" "${control_roots[@]}" &
lift_into nexus-close "$here/gen/model-scala.jar=$scala/" \
  "${nexus_close_roots[@]/#/temporal.nexuscaller.closepolicy.Claims\$package\$.}" &
wait
for name in presence channels declarations admission closereset realizations activity activity-system activity-race nexus-close nexus-control; do
  [[ "$(cat "$here/gen/lifts/$name.status")" == 0 ]] \
    || { grep -v '^WARNING' "$here/gen/lifts/$name.log" >&2; echo "run.sh: the $name fixture did not lift" >&2; exit 1; }
done
grep '^lift:' "$here/gen/lifts/rejects.log" > "$here/gen/lifts/rejects.txt" || true
[[ "$(cat "$here/gen/lifts/rejects.status")" != 0 && ! -f "$here/gen/lifts/rejects.json" ]] \
  || { echo "run.sh: the lifter wrote the rejected declarations' IR" >&2; exit 1; }
for name in presence channels declarations admission closereset realizations rejects activity activity-system activity-race nexus-close nexus-control; do
  file="$name.json"
  [[ "$name" == rejects ]] && file=rejects.txt
  # The activity Models' IR and the Nexus close designs' are checked in beside the Nexus caller's; the
  # fixtures' beside their sources.
  expected="lifter/testdata/lifts/expected/$file"
  [[ "$name" == activity* || "$name" == nexus-close || "$name" == nexus-control ]] && expected="ir/$file"
  if $update; then
    cp "$here/gen/lifts/$file" "$here/$expected"
  elif ! diff -q "$here/$expected" "$here/gen/lifts/$file" >/dev/null 2>&1; then
    diff "$here/$expected" "$here/gen/lifts/$file" | head -20 || true
    echo "run.sh: $expected is stale; rerun with --update" >&2
    exit 1
  fi
done
echo "lifted presence, channels, declarations, admission, closereset, realizations, the activity Models and the Nexus close designs; refused $(wc -l < "$here/gen/lifts/rejects.txt" | tr -d ' ') declarations"

echo "== a declaration's identity does not move with its line"
shifted="$(mktemp -d)"
{ printf '// A line the declarations move down by.\n%.0s' 1 2 3; cat "$here/gen/fixtures/lifts/Declarations.scala"; } \
  > "$shifted/Declarations.scala"
cp "$here/gen/fixtures/lifts/project.scala" "$shifted/project.scala"
scala_build --power package --library "$shifted" -f -o "$here/gen/shifted.jar"
lift "$here/gen/shifted.jar=$(stored lifts),$here/gen/model-scala.jar=$scala/" \
  "$here/gen/model-scala.classpath" "$here/gen/lifts/shifted.json" 'fixture.declarations.Declarations$package$.queries' \
  'fixture.declarations.Declarations$package$.durableEventually'
rm -rf "$shifted"
lines() { sed -E 's/"line": [0-9]+/"line": _/' "$1"; }
cmp -s "$here/gen/lifts/declarations.json" "$here/gen/lifts/shifted.json" \
  && { echo "run.sh: moving the declarations did not move their lines" >&2; exit 1; }
diff <(lines "$here/gen/lifts/declarations.json") <(lines "$here/gen/lifts/shifted.json") \
  || { echo "run.sh: moving the declarations down changed more than their lines" >&2; exit 1; }
echo "moved down 3 lines: only the lines changed"

echo "== generate or check every lowered Case and the Query manifest"
case_args=()
$update && case_args+=(--update)
go run ./tools/umpire/cmd/umpire-gen-cases ${case_args[@]+"${case_args[@]}"}

echo "== interpret the IR in Go and compare with Lean"
go vet ./model/scalav2/...
go test -count=1 ./model/scalav2/...
echo "== ok"
