#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$script_dir/testlib.sh"

test_tmp=$(mktemp -d)
cleanup() {
	rm -rf "$test_tmp"
}
trap cleanup EXIT

fail() {
	printf '%s\n' "$1" >&2
	exit 1
}

require_file() {
	local path=$1
	local expected=$2
	local actual
	actual=$(<"$path")
	if [[ "$actual" != "$expected" ]]; then
		printf 'unexpected contents in %s: got %q, want %q\n' "$path" "$actual" "$expected" >&2
		exit 1
	fi
}

success_dir="$test_tmp/success"
gomad_run_checked 5 0 'success fixture seed=1 mode=unit iteration=0' "$success_dir" -- \
	bash -c 'printf "standard output\n"; printf "standard error\n" >&2'
require_file "$success_dir/stdout" 'standard output'
require_file "$success_dir/stderr" 'standard error'
require_file "$success_dir/status" '0'
require_file "$success_dir/timed-out" '0'

expected_failure_dir="$test_tmp/expected-failure"
gomad_run_checked 5 7 'expected failure seed=2 mode=unit iteration=1' "$expected_failure_dir" -- \
	bash -c 'printf "failed output\n"; printf "failed error\n" >&2; exit 7'
require_file "$expected_failure_dir/stdout" 'failed output'
require_file "$expected_failure_dir/stderr" 'failed error'
require_file "$expected_failure_dir/status" '7'
require_file "$expected_failure_dir/timed-out" '0'

unexpected_dir="$test_tmp/unexpected"
if gomad_run_checked 5 0 'unexpected fixture seed=7 mode=unit iteration=3' "$unexpected_dir" -- \
	bash -c 'printf "unexpected output\n"; printf "unexpected error\n" >&2; exit 9' \
	2>"$test_tmp/unexpected-diagnostic"; then
	fail 'checked runner accepted an unexpected exit status'
fi
require_file "$unexpected_dir/status" '9'
if ! grep -Fq 'gomadv3 process failed: unexpected fixture seed=7 mode=unit iteration=3: status 9, want 0' "$test_tmp/unexpected-diagnostic"; then
	fail 'checked runner diagnostic omitted run metadata or status'
fi
if ! grep -Fq 'unexpected output' "$test_tmp/unexpected-diagnostic" || \
	! grep -Fq 'unexpected error' "$test_tmp/unexpected-diagnostic"; then
	fail 'checked runner diagnostic omitted captured output'
fi

timeout_dir="$test_tmp/timeout"
descendant_pid_file="$test_tmp/descendant-pid"
gomad_run_checked 1 124 'timeout fixture seed=11 mode=unit iteration=4' "$timeout_dir" -- \
	bash -c 'sleep 30 & printf "%s\n" "$!" >"$1"; wait' bash "$descendant_pid_file"
require_file "$timeout_dir/status" '124'
require_file "$timeout_dir/timed-out" '1'
descendant_pid=$(<"$descendant_pid_file")
for _ in {1..20}; do
	if ! kill -0 "$descendant_pid" 2>/dev/null; then
		descendant_pid=
		break
	fi
	sleep 0.1
done
if [[ -n "$descendant_pid" ]]; then
	fail "checked runner left descendant $descendant_pid running after timeout"
fi

false_timeout_dir="$test_tmp/false-timeout"
if gomad_run_checked 5 124 'false timeout seed=13 mode=unit iteration=5' "$false_timeout_dir" -- \
	bash -c 'exit 124' 2>"$test_tmp/false-timeout-diagnostic"; then
	fail 'checked runner accepted exit status 124 as a watchdog timeout'
fi
require_file "$false_timeout_dir/status" '124'
require_file "$false_timeout_dir/timed-out" '0'
if ! grep -Fq 'gomadv3 process failed: false timeout seed=13 mode=unit iteration=5: status 124 was not a timeout' \
	"$test_tmp/false-timeout-diagnostic"; then
	fail 'checked runner false-timeout diagnostic omitted run metadata'
fi

parallel_success_dir="$test_tmp/parallel-success"
parallel_failure_dir="$test_tmp/parallel-failure"
gomad_run_checked 5 0 'parallel success seed=3 mode=unit iteration=6' "$parallel_success_dir" -- \
	bash -c 'printf "parallel success\n"' &
parallel_success_pid=$!
gomad_run_checked 5 0 'parallel failure seed=5 mode=unit iteration=7' "$parallel_failure_dir" -- \
	bash -c 'exit 23' 2>"$test_tmp/parallel-failure-diagnostic" &
parallel_failure_pid=$!
if ! wait "$parallel_success_pid"; then
	fail 'successful parallel child failed'
fi
if wait "$parallel_failure_pid"; then
	fail 'failed parallel child reported success'
fi
require_file "$parallel_success_dir/status" '0'
require_file "$parallel_failure_dir/status" '23'

printf 'gomadv3 checked runner tests passed\n'
