#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
repo=$(cd .. && pwd -P)
plugin="$repo/.bin/protoc-gen-go-v1.36.6"
grpc_plugin="$repo/.bin/protoc-gen-go-grpc-v1.3.0"
for executable in "$plugin" "$grpc_plugin"; do
  if [ ! -x "$executable" ]; then
    printf 'missing pinned generator; run make %s from temporal\n' "${executable#"$repo/"}" >&2
    exit 1
  fi
done
case "$("$plugin" --version)" in
  'protoc-gen-go v1.36.6'|'protoc-gen-go-v1.36.6 v1.36.6') ;;
  *) printf 'wrong protoc-gen-go version\n' >&2; exit 1 ;;
esac
[ "$("$grpc_plugin" --version)" = 'protoc-gen-go-grpc 1.3.0' ] || { printf 'wrong protoc-gen-go-grpc version\n' >&2; exit 1; }
[ "$(mise exec -- protoc --version)" = 'libprotoc 29.5' ] || { printf 'wrong protoc version; expected 29.5\n' >&2; exit 1; }
output="$PWD/umpire/reference/kettle/gen"
if [ "$#" -eq 1 ] && [ "$1" = --check ]; then
  output=$(mktemp -d)
  trap 'rm -rf "${output:?}"' EXIT
elif [ "$#" -ne 0 ]; then
  printf 'usage: proto-kettle-go.sh [--check]\n' >&2
  exit 1
fi
mise exec -- protoc --plugin="protoc-gen-go=$plugin" --plugin="protoc-gen-go-grpc=$grpc_plugin" \
  --proto_path=umpire/reference/kettle/proto --proto_path=umpire/proto --go_out="$output" --go_opt=paths=source_relative \
  --go-grpc_out="$output" --go-grpc_opt=paths=source_relative \
  umpire/reference/kettle/proto/kettle/v1/kettle.proto \
  umpire/reference/kettle/proto/kettle/v1/probe.proto
if [ "${1:-}" = --check ]; then
  diff -ru umpire/reference/kettle/gen "$output"
fi
