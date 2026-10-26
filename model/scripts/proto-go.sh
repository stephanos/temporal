#!/usr/bin/env bash
set -euo pipefail
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
cd "$(dirname "$0")/.."
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ] || { [ "$#" -eq 2 ] && [ "$2" != --check ]; }; then
  printf 'usage: proto-go.sh <temporal> [--check]\n' >&2
  exit 1
fi
module=$1
includes=(--proto_path=umpire/protocol/proto)
case "$module" in
  temporal)
    destination=temporal/hosts/go
    includes=(--proto_path=temporal/proto "${includes[@]}")
    inputs=(temporal/proto/umpire/temporal/v1/{temporal,probe,activity}.proto)
    ;;
  *) printf 'usage: proto-go.sh <temporal> [--check]\n' >&2; exit 1 ;;
esac
repo=$(cd .. && pwd -P)
plugin="$repo/.bin/protoc-gen-go-v1.36.6"
plugins=(--plugin="protoc-gen-go=$plugin")
if [ ! -x "$plugin" ]; then
  printf 'missing pinned protoc-gen-go v1.36.6; run make .bin/protoc-gen-go-v1.36.6 from temporal\n' >&2
  exit 1
fi
case "$("$plugin" --version)" in
  'protoc-gen-go v1.36.6'|'protoc-gen-go-v1.36.6 v1.36.6') ;;
  *) printf 'wrong protoc-gen-go version: %s\n' "$plugin" >&2; exit 1 ;;
esac
output="$PWD/$destination"
if [ "${2:-}" = --check ]; then
  temporary=$(mktemp -d)
  output=$temporary
  trap 'rm -rf "${temporary:?}"' EXIT
fi
options=(--go_out="$output" --go_opt=paths=source_relative)
mise exec -- protoc "${plugins[@]}" "${includes[@]}" "${options[@]}" "${inputs[@]}"
if [ "${2:-}" = --check ]; then
  destination+=/umpire
  output+=/umpire
  linked=$(find "$destination" -type l -print -quit)
  if [ -L "$destination" ] || [ -n "$linked" ]; then
    printf 'not generated: %s\n' "${linked:-$destination}" >&2
    exit 1
  fi
  diff -ru "$destination" "$output"
fi
