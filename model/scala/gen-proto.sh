#!/usr/bin/env bash
# Generate Java classes for the Testpilot Case and Run protos and their import closure, and package
# them as gen/testpilot-proto.jar for the Case producer. The API protos come from proto/api.binpb,
# the same descriptor set the Lean workspace's Testpilot modules read.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
gen="$here/gen"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

protoc --proto_path="$root/proto/internal" --descriptor_set_in="$root/proto/api.binpb" \
  --include_imports --descriptor_set_out="$scratch/set.binpb" \
  "$root/proto/internal/temporal/server/api/testpilot/v1/case.proto" \
  "$root/proto/internal/temporal/server/api/testpilot/v1/run.proto"

include="$(dirname "$(dirname "$(command -v protoc)")")/include"
files=$(protoc -I"$include" --decode=google.protobuf.FileDescriptorSet google/protobuf/descriptor.proto \
  < "$scratch/set.binpb" | sed -n 's/^  name: "\(.*\)"$/\1/p' | grep -v '^google/protobuf/')
mkdir -p "$scratch/java"
# shellcheck disable=SC2086
protoc --descriptor_set_in="$scratch/set.binpb" --java_out="$scratch/java" $files

mkdir -p "$gen"
mise exec -- scala-cli --power package --library "$scratch/java" \
  --dep com.google.protobuf:protobuf-java:4.29.5 --jvm 27 -f -o "$gen/testpilot-proto.jar" >/dev/null
# The stamp lets run.sh regenerate only when a proto or the descriptor set changed.
cat "$root"/proto/internal/temporal/server/api/testpilot/v1/*.proto "$root/proto/api.binpb" \
  | shasum -a 256 | cut -d' ' -f1 > "$gen/testpilot-proto.stamp"
echo "wrote $gen/testpilot-proto.jar"
