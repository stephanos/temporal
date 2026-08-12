#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
go_bin="$script_dir/.toolchain/bin/go"
test_tmp=$(mktemp -d "$script_dir/.toolchain/intercept-failure.XXXXXX")
cleanup() {
	find "$test_tmp" -depth -delete
}
trap cleanup EXIT

require_failure() {
	local package=$1
	local diagnostic=$2
	local name=${package##*/}
	if env -u GOMADSEED -u GOMADV3_CHILD_SEED GOCACHE="$test_tmp/cache-$name" GOWORK=off \
		"$go_bin" -C "$script_dir/testdata" test "./interceptfail/$package" \
		>"$test_tmp/$name.stdout" 2>"$test_tmp/$name.stderr"; then
		printf 'gomadv3 compiler accepted invalid interception fixture: %s\n' "$package" >&2
		exit 1
	fi
	if ! grep -Fq "$diagnostic" "$test_tmp/$name.stderr"; then
		printf 'gomadv3 compiler omitted interception diagnostic for %s\n' "$package" >&2
		cat "$test_tmp/$name.stderr" >&2
		exit 1
	fi
}

require_failure missing_target 'gomad interception target is missing: Target'
require_failure missing_hook 'gomad interception hook is missing: Hook'
require_failure bad_parameter 'gomad interception signature mismatch for Target: hook parameter 1'
require_failure bad_result 'gomad interception signature mismatch for Target: hook result 1'
require_failure bad_handled 'gomad interception signature mismatch for Target: hook final result must be bool'
require_failure duplicate_target 'gomad interception target is duplicated in manifest: Target and Target'
require_failure bodyless_target 'missing function body'
require_failure variadic 'gomad interception signature mismatch for Target: hook variadic form does not match target'

printf 'gomadv3 interception failure tests passed\n'
