#!/usr/bin/env bash
# The model/scalav2 gate. Scala is only the authoring front end: model/scala's Models are compiled,
# the lifter reads their typed trees and emits the IR, and Go interprets the IR and checks every
# table, identity and fingerprint against the Lean Model. Exits non-zero on the first failure.
#
#   model/scalav2/run.sh           lift, require ir/nexus-caller.json to be current, test
#   model/scalav2/run.sh --update  lift and rewrite ir/nexus-caller.json
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
roots=('temporal.nexuscaller.Model$package$.nexusProduct' 'temporal.nexuscaller.Model$package$.nexusProtocol'
  'temporal.nexuscaller.Model$package$.handlerWorker' 'temporal.worker.Worker$package$.polling')

schema="$root/proto/internal/temporal/server/api/umpire/v1/ir.proto"
stamp="$(shasum -a 256 "$schema" | cut -d' ' -f1)"
if [[ ! -f "$here/gen/ir-proto.jar" || "$(cat "$here/gen/ir.stamp" 2>/dev/null)" != "$stamp" ]]; then
  echo "== generate the IR's Java classes"
  "$here/gen.sh"
  echo "$stamp" > "$here/gen/ir.stamp"
fi
[[ api/umpire/v1/ir.pb.go -nt "$schema" ]] || { echo "run.sh: api/umpire/v1 is older than the IR schema; run make protoc" >&2; exit 1; }

echo "== compile model/scala and package its TASTy"
scala_cli --power package --library model/scala/project.scala model/scala/umpire model/scala/temporal \
  -f -o "$here/gen/model-scala.jar" >/dev/null
scala_cli compile --print-class-path model/scala/project.scala model/scala/umpire model/scala/temporal \
  > "$here/gen/model-scala.classpath"

echo "== lift the Nexus caller Model"
lifted="$(mktemp)"
scala_cli run "$here/lifter" -- "$here/gen/model-scala.jar" "$here/gen/model-scala.classpath" "$lifted" model/scala/ "${roots[@]}" \
  2> >(grep -v '^WARNING' >&2)
if $update; then
  cp "$lifted" "$here/ir/nexus-caller.json"
elif ! diff -q "$here/ir/nexus-caller.json" "$lifted" >/dev/null; then
  diff "$here/ir/nexus-caller.json" "$lifted" | head -20
  echo "run.sh: ir/nexus-caller.json is stale; rerun with --update" >&2
  exit 1
fi
rm -f "$lifted"

echo "== the lifter refuses a construct outside the subset, at its line"
fixture="$here/lifter/testdata/unsupported"
scala_cli --power package --library "$fixture" -f -o "$here/gen/unsupported.jar" >/dev/null
refused="$(scala_cli run "$here/lifter" -- "$here/gen/unsupported.jar" "$here/gen/model-scala.classpath" /dev/null \
  model/scalav2/lifter/testdata/unsupported/ 'temporal.fixture.Unsupported$package$.unsupported' 2>&1 | grep '^lift:' || true)"
expected='lift: model/scalav2/lifter/testdata/unsupported/Unsupported.scala:18: `var out` has no IR form'
[[ "$refused" == "$expected"* ]] || { echo "run.sh: expected '$expected ...', got '$refused'" >&2; exit 1; }
echo "$refused"

echo "== interpret the IR in Go and compare with Lean"
go vet ./model/scalav2/...
go test -count=1 ./model/scalav2/...
echo "== ok"
