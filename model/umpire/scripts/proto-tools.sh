#!/usr/bin/env bash
set -euo pipefail
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
cd "$(dirname "$0")/.."
root=$(pwd -P)
if [ -L .bin ]; then
  printf 'generator directory must not be a symlink\n' >&2
  exit 1
fi
mkdir -p .bin
[ "$(cd .bin && pwd -P)" = "$root/.bin" ] || exit 1
lock="$root/.bin/proto-tools.lock"
if ! mkdir "$lock"; then
  printf 'generator installation is locked: %s\n' "$lock" >&2
  exit 1
fi
trap 'rm -rf "${lock:?}"' EXIT
for specification in \
  'protoc-gen-go:v1.36.6:google.golang.org/protobuf/cmd/protoc-gen-go:protoc-gen-go v1.36.6' \
  'protoc-gen-go-grpc:v1.3.0:google.golang.org/grpc/cmd/protoc-gen-go-grpc:protoc-gen-go-grpc 1.3.0'; do
  name=${specification%%:*}; rest=${specification#*:}
  version=${rest%%:*}; rest=${rest#*:}
  package=${rest%%:*}; expected=${rest#*:}
  destination="$root/.bin/$name-$version"
  if [ -L "$destination" ]; then
    printf 'generator must not be a symlink: %s\n' "$destination" >&2
    exit 1
  fi
  if [ ! -e "$destination" ]; then
    GOBIN="$lock" mise exec -- go install "$package@$version"
    [ -f "$lock/$name" ] && [ ! -L "$lock/$name" ] && [ -x "$lock/$name" ]
    [ "$("$lock/$name" --version)" = "$expected" ]
    [ ! -L .bin ] && [ "$(cd .bin && pwd -P)" = "$root/.bin" ]
    [ ! -e "$destination" ] && [ ! -L "$destination" ]
    mv "$lock/$name" "$destination"
  fi
  case "$("$destination" --version)" in
    "$expected"|"$name-$version ${version#v}") ;;
    *) printf 'wrong pinned generator: %s\n' "$destination" >&2; exit 1 ;;
  esac
done
