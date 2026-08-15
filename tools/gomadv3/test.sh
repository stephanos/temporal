#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
host_go=${GOMADV3_BOOTSTRAP_GO:-}
if [ -z "$host_go" ]; then
	host_go=$(command -v go || true)
fi
if [ -z "$host_go" ] || [ ! -x "$host_go" ]; then
	printf 'gomadv3 requires an installed bootstrap Go; set GOMADV3_BOOTSTRAP_GO\n' >&2
	exit 1
fi

hosttool="$script_dir/.toolchain/hosttool"
mkdir -p "$script_dir/.toolchain"
unset GOMADSEED GOMADV3_CHILD_SEED
GOCACHE="$script_dir/.toolchain/generator-cache" GOTOOLCHAIN=local GOWORK=off \
	"$host_go" -C "$script_dir" build -o "$hosttool" ./toolchain/cmd/gomadtool

case ${1:-test} in
--list-tiers)
	exec "$hosttool" test-mode --mode="${2:-test}" --output=tiers
	;;
--success-message)
	exec "$hosttool" test-mode --mode="${2:-}" --output=success
	;;
validate)
	exec "$hosttool" patch-validate --root="$script_dir"
	;;
esac

go_bin="$script_dir/.toolchain/bin/go"
if [ ! -x "$go_bin" ]; then
	printf 'gomadv3 toolchain is missing: run make -C %s toolchain\n' "$script_dir" >&2
	exit 1
fi
exec "$hosttool" test --root="$script_dir" --mode="${1:-test}" --go="$go_bin"
