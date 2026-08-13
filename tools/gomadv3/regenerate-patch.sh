#!/bin/sh

set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
if [ "$#" -ne 1 ]; then
	printf 'usage: %s GO-SOURCE-ROOT\n' "$0" >&2
	exit 1
fi
bootstrap_go=${GOMADV3_BOOTSTRAP_GO:-}
if [ -z "$bootstrap_go" ]; then
	bootstrap_go=$(command -v go || true)
fi
if [ -z "$bootstrap_go" ] || [ ! -x "$bootstrap_go" ]; then
	printf 'gomadv3 requires an installed bootstrap Go; set GOMADV3_BOOTSTRAP_GO\n' >&2
	exit 1
fi
set -- patch-regenerate --root="$script_dir" --candidate-root="$1" --bootstrap-go="$bootstrap_go"
if [ -n "${GOMADV3_PATCH_OUTPUT:-}" ]; then
	set -- "$@" --output="$GOMADV3_PATCH_OUTPUT"
fi
unset GOMADSEED
GOCACHE="$script_dir/.toolchain/generator-cache" GOTOOLCHAIN=local GOWORK=off \
	exec "$bootstrap_go" -C "$script_dir" run ./internal/hosttool "$@"
