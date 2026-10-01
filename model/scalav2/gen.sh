#!/usr/bin/env bash
# Package the Java classes the Scala sources compile against:
#
#   gen/ir-proto.jar         the IR's classes, for the Scala lifter. The IR schema is
#                            proto/internal/temporal/server/api/modelir/v1/ir.proto; its Go code is
#                            generated with every other internal proto by `make protoc` into
#                            api/modelir/v1.
#
#   gen.sh [ir]...            package the named jars, every one when none is named
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

jars=()
for arg in "$@"; do
  case "$arg" in
    ir) jars+=("$arg") ;;
    --if-stale) for jar in ir; do stale "$jar" && jars+=("$jar"); done ;;
    *) echo "usage: gen.sh [ir]... | gen.sh --if-stale" >&2; exit 2 ;;
  esac
done
[[ $# -gt 0 ]] || jars=(ir)

mkdir -p "$gen"
for jar in ${jars[@]+"${jars[@]}"}; do
  "$jar"
  stamp "$jar" > "$gen/$jar.stamp"
  echo "generated gen/$jar-proto.jar"
done
