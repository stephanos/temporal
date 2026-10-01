#!/usr/bin/env bash
# Package the Java classes the Scala sources compile against:
#
#   gen/ir-proto.jar         the IR's classes, for the Scala lifter. The IR schema is
#                            proto/internal/temporal/server/api/modelir/v1/ir.proto; its Go code is
#                            generated with every other internal proto by `make protoc` into
#                            api/modelir/v1.
#   gen/testpilot-proto.jar  the Testpilot Case and Run protos and their import closure, for the Case
#                            producer in scala/umpire/caseproducer. The API protos come from
#                            proto/api.binpb, the same descriptor set the Lean workspace's Testpilot
#                            modules read.
#
#   gen.sh [ir|testpilot]...  package the named jars, both when none is named
#   gen.sh --if-stale         package only the jars whose inputs changed since they were packaged
#
# Each jar has a stamp beside it, the hash of its inputs. protoc and the JDK come from mise.toml.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
gen="$here/gen"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

ir_schema="$root/proto/internal/temporal/server/api/modelir/v1/ir.proto"
stamp() {
  case "$1" in
    ir) shasum -a 256 "$ir_schema" | cut -d' ' -f1 ;;
    testpilot) cat "$root"/proto/internal/temporal/server/api/testpilot/v1/*.proto "$root/proto/api.binpb" \
      | shasum -a 256 | cut -d' ' -f1 ;;
  esac
}
stale() { [[ ! -f "$gen/$1-proto.jar" || "$(cat "$gen/$1.stamp" 2>/dev/null)" != "$(stamp "$1")" ]]; }

ir() {
  mkdir -p "$scratch/ir"
  mise exec -- protoc --proto_path="$root/proto/internal" --java_out="$scratch/ir" \
    temporal/server/api/modelir/v1/ir.proto
  mise exec -- scala-cli --power package --library "$scratch/ir" \
    --dep com.google.protobuf:protobuf-java:4.29.5 -f -o "$gen/ir-proto.jar" \
    --suppress-outdated-dependency-warning >/dev/null
}

testpilot() {
  local protoc include files
  protoc="$(mise which protoc)"
  include="$(dirname "$(dirname "$protoc")")/include"
  "$protoc" --proto_path="$root/proto/internal" --descriptor_set_in="$root/proto/api.binpb" \
    --include_imports --descriptor_set_out="$scratch/set.binpb" \
    "$root/proto/internal/temporal/server/api/testpilot/v1/case.proto" \
    "$root/proto/internal/temporal/server/api/testpilot/v1/run.proto"
  files=$("$protoc" -I"$include" --decode=google.protobuf.FileDescriptorSet google/protobuf/descriptor.proto \
    < "$scratch/set.binpb" | sed -n 's/^  name: "\(.*\)"$/\1/p' | grep -v '^google/protobuf/')
  mkdir -p "$scratch/testpilot"
  # shellcheck disable=SC2086
  "$protoc" --descriptor_set_in="$scratch/set.binpb" --java_out="$scratch/testpilot" $files
  mise exec -- scala-cli --power package --library "$scratch/testpilot" \
    --dep com.google.protobuf:protobuf-java:4.29.5 --jvm 27 -f -o "$gen/testpilot-proto.jar" \
    --suppress-outdated-dependency-warning >/dev/null
}

jars=()
for arg in "$@"; do
  case "$arg" in
    ir | testpilot) jars+=("$arg") ;;
    --if-stale) for jar in ir testpilot; do stale "$jar" && jars+=("$jar"); done ;;
    *) echo "usage: gen.sh [ir|testpilot]... | gen.sh --if-stale" >&2; exit 2 ;;
  esac
done
[[ $# -gt 0 ]] || jars=(ir testpilot)

mkdir -p "$gen"
for jar in ${jars[@]+"${jars[@]}"}; do
  "$jar"
  stamp "$jar" > "$gen/$jar.stamp"
  echo "generated gen/$jar-proto.jar"
done
