#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)

if ! make --no-print-directory -n -C "$repo_root" gomadv3 >/dev/null; then
	printf 'root gomadv3 target does not resolve to the argv-safe CLI build\n' >&2
	exit 1
fi

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

success_messages=$(
	"$script_dir/test.sh" --success-message test
	"$script_dir/test.sh" --success-message test-builder
	"$script_dir/test.sh" --success-message test-runtime
	"$script_dir/test.sh" --success-message test-upstream
)
if [[ "$success_messages" != $'gomadv3 all black-box tiers passed\ngomadv3 builder tier passed\ngomadv3 runtime tier passed\ngomadv3 upstream-compatibility tier passed' ]]; then
	printf 'gomadv3 test tier success messages are incomplete or ambiguous\n' >&2
	exit 1
fi

printf 'gomadv3 test mode tests passed\n'
