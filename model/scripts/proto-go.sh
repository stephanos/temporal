#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ] || { [ "$#" -eq 2 ] && [ "$2" != --check ]; }; then
  printf 'usage: proto-go.sh <core|kettle|temporal> [--check]\n' >&2
  exit 1
fi
module=$1
includes=(--proto_path=umpire/protocol/proto)
case "$module" in
  core)
    destination=umpire/hosts/go/gen
    inputs=(umpire/protocol/proto/umpire/{model/v1/model,realization/v1/realization,annotations/v1/annotations,run/v1/run}.proto)
    ;;
  kettle)
    destination=umpire/reference/kettle/gen
    includes=(--proto_path=umpire/reference/kettle/proto "${includes[@]}")
    inputs=(umpire/reference/kettle/proto/kettle/v1/{kettle,probe}.proto)
    ;;
  temporal)
    destination=temporal/hosts/go
    includes=(--proto_path=temporal/proto "${includes[@]}")
    inputs=(temporal/proto/umpire/temporal/v1/{temporal,probe,activity}.proto)
    ;;
  *) printf 'usage: proto-go.sh <core|kettle|temporal> [--check]\n' >&2; exit 1 ;;
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
if [ "$module" = kettle ]; then
  grpc_plugin="$repo/.bin/protoc-gen-go-grpc-v1.3.0"
  if [ ! -x "$grpc_plugin" ]; then
    printf 'missing pinned generator; run make .bin/protoc-gen-go-grpc-v1.3.0 from temporal\n' >&2
    exit 1
  fi
  if [ "$("$grpc_plugin" --version)" != 'protoc-gen-go-grpc 1.3.0' ]; then
    printf 'wrong protoc-gen-go-grpc version\n' >&2
    exit 1
  fi
  plugins+=(--plugin="protoc-gen-go-grpc=$grpc_plugin")
fi
if [ "$(mise exec -- protoc --version)" != 'libprotoc 29.5' ]; then
  printf 'wrong protoc version; expected mise-pinned libprotoc 29.5\n' >&2
  exit 1
fi
output="$PWD/$destination"
if [ "${2:-}" = --check ]; then
  temporary=$(mktemp -d)
  output=$temporary
  trap 'rm -rf "${temporary:?}"' EXIT
fi
options=(--go_out="$output" --go_opt=paths=source_relative)
if [ "$module" = kettle ]; then
  options+=(--go-grpc_out="$output" --go-grpc_opt=paths=source_relative)
fi
mise exec -- protoc "${plugins[@]}" "${includes[@]}" "${options[@]}" "${inputs[@]}"
if [ "${2:-}" = --check ]; then
  if [ "$module" = temporal ]; then
    destination+=/umpire
    output+=/umpire
  fi
  linked=$(find "$destination" -type l -print -quit)
  if [ -L "$destination" ] || [ -n "$linked" ]; then
    printf 'not generated: %s\n' "${linked:-$destination}" >&2
    exit 1
  fi
  diff -ru "$destination" "$output"
fi
