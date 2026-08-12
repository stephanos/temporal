#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

all_tiers=$("$script_dir/test.sh" --list-tiers test)
individual_tiers=$(
	"$script_dir/test.sh" --list-tiers test-builder
	"$script_dir/test.sh" --list-tiers test-runtime
	"$script_dir/test.sh" --list-tiers test-upstream
)

if [[ "$all_tiers" != "$individual_tiers" ]]; then
	printf 'gomadv3 full test tier order differs from the focused tier order\n' >&2
	exit 1
fi

if [[ "$all_tiers" != $'test-builder\ntest-runtime\ntest-upstream' ]]; then
	printf 'gomadv3 test tiers are incomplete or reordered\n' >&2
	exit 1
fi

printf 'gomadv3 test mode tests passed\n'
