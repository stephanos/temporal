#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
repo=$(cd .. && pwd -P)
plugin="$repo/.bin/protoc-gen-go-v1.36.6"
if [ ! -x "$plugin" ]; then
  printf 'missing pinned protoc-gen-go v1.36.6; run make .bin/protoc-gen-go-v1.36.6 from temporal\n' >&2
  exit 1
fi
case "$("$plugin" --version)" in
  'protoc-gen-go v1.36.6'|'protoc-gen-go-v1.36.6 v1.36.6') ;;
  *) printf 'wrong protoc-gen-go version: %s\n' "$plugin" >&2; exit 1 ;;
esac
if [ "$(mise exec -- protoc --version)" != 'libprotoc 29.5' ]; then
  printf 'wrong protoc version; expected mise-pinned libprotoc 29.5\n' >&2
  exit 1
fi
output="$PWD/temporal/hosts/go"
if [ "${1:-}" = --check ]; then
  output=$(mktemp -d)
  trap 'rm -rf "$output"' EXIT
elif [ "$#" -ne 0 ]; then
  printf 'usage: proto-temporal-go.sh [--check]\n' >&2
  exit 1
fi
mise exec -- protoc --plugin="protoc-gen-go=$plugin" --proto_path=temporal/proto --proto_path=umpire/protocol/proto \
  --go_out="$output" --go_opt=paths=source_relative \
  temporal/proto/umpire/temporal/v1/temporal.proto temporal/proto/umpire/temporal/v1/probe.proto temporal/proto/umpire/temporal/v1/activity.proto
if [ "${1:-}" = --check ]; then
  diff -ru temporal/hosts/go/umpire "$output/umpire"
fi
