#!/bin/sh

set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
host_go=$(command -v go || true)
if [ -z "$host_go" ] || [ ! -x "$host_go" ]; then
	printf 'gomadv3 host tooling requires an installed Go command\n' >&2
	exit 1
fi
bootstrap_go=${GOMADV3_BOOTSTRAP_GO:-$host_go}
build_bash=$(command -v bash || true)
if [ -z "$build_bash" ] || [ ! -x "$build_bash" ]; then
	printf 'gomadv3 requires Bash to run upstream make.bash\n' >&2
	exit 1
fi
unset GOMADSEED GOMADV3_CHILD_SEED
hosttool_bin="$script_dir/.toolchain/hosttool"
GOCACHE="$script_dir/.toolchain/generator-cache" GOTOOLCHAIN=local GOWORK=off \
"$host_go" -C "$script_dir" build -o "$hosttool_bin" ./toolchain/cmd/gomadtool
exec "$hosttool_bin" toolchain-build \
	--root="$script_dir" --bootstrap-go="$bootstrap_go" --build-bash="$build_bash"
