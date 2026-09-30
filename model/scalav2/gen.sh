#!/usr/bin/env bash
# Package the IR's Java classes as gen/ir-proto.jar for the Scala lifter. The IR schema is
# proto/internal/temporal/server/api/umpire/v1/ir.proto; its Go code is generated with every other
# internal proto by `make protoc` into api/umpire/v1. protoc and the JDK come from mise.toml.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

mkdir -p "$scratch/java" "$here/gen"
mise exec -- protoc --proto_path="$root/proto/internal" --java_out="$scratch/java" \
  temporal/server/api/umpire/v1/ir.proto
mise exec -- scala-cli --power package --library "$scratch/java" \
  --dep com.google.protobuf:protobuf-java:4.29.5 -f -o "$here/gen/ir-proto.jar" \
  --suppress-outdated-dependency-warning >/dev/null
echo "generated gen/ir-proto.jar"
