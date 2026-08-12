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

bounded_output_dir="$test_tmp/bounded-output"
gomad_run_checked 5 0 'bounded output seed=17 mode=unit iteration=6' "$bounded_output_dir" -- \
	perl -e 'print "o" x (2 * 1024 * 1024); print STDERR "e" x (2 * 1024 * 1024)'
if [[ $(wc -c <"$bounded_output_dir/stdout") -ne 1048576 ]] || \
	[[ $(wc -c <"$bounded_output_dir/stderr") -ne 1048576 ]]; then
	fail 'checked runner did not enforce its per-stream output bound'
fi
require_file "$bounded_output_dir/output-truncated" '1'

bounded_timeout_dir="$test_tmp/bounded-timeout"
gomad_run_checked 1 124 'bounded timeout seed=19 mode=unit iteration=7' "$bounded_timeout_dir" -- \
	perl -e '$| = 1; print "o" x (2 * 1024 * 1024); print STDERR "e" x (2 * 1024 * 1024); sleep 30'
if [[ $(wc -c <"$bounded_timeout_dir/stdout") -ne 1048576 ]] || \
	[[ $(wc -c <"$bounded_timeout_dir/stderr") -ne 1048576 ]]; then
	fail 'checked runner did not preserve its output bound on timeout'
fi
require_file "$bounded_timeout_dir/timed-out" '1'
require_file "$bounded_timeout_dir/output-truncated" '1'

parallel_success_dir="$test_tmp/parallel-success"
parallel_failure_dir="$test_tmp/parallel-failure"
gomad_run_checked 5 0 'parallel success seed=3 mode=unit iteration=8' "$parallel_success_dir" -- \
	bash -c 'printf "parallel success\n"' &
parallel_success_pid=$!
gomad_run_checked 5 0 'parallel failure seed=5 mode=unit iteration=9' "$parallel_failure_dir" -- \
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

exec_wrapper="$script_dir/exec.sh"

missing_seed_dir="$test_tmp/missing-seed"
gomad_run_checked 5 125 'exec wrapper seed=unset mode=missing-seed iteration=0' "$missing_seed_dir" -- \
	env -u GOMADV3_CHILD_SEED "$exec_wrapper" true
if ! grep -Fq 'gomadv3 exec: GOMADV3_CHILD_SEED is required' "$missing_seed_dir/stderr"; then
	fail 'exec wrapper missing-seed diagnostic is absent'
fi

missing_command_dir="$test_tmp/missing-command"
gomad_run_checked 5 125 'exec wrapper seed=1 mode=missing-command iteration=0' "$missing_command_dir" -- \
	env GOMADV3_CHILD_SEED=1 "$exec_wrapper"
if ! grep -Fq 'gomadv3 exec: target command is required' "$missing_command_dir/stderr"; then
	fail 'exec wrapper missing-command diagnostic is absent'
fi

arguments_dir="$test_tmp/arguments"
gomad_run_checked 5 0 'exec wrapper seed=max mode=arguments iteration=0' "$arguments_dir" -- \
	env GOMADSEED=inherited GOMADV3_CHILD_SEED=18446744073709551615 "$exec_wrapper" \
	bash -c 'printf "seed=%s child=%s arg1=%s arg2=%s\n" "$GOMADSEED" "${GOMADV3_CHILD_SEED-unset}" "$1" "$2"' \
	bash 'two words' '*'
require_file "$arguments_dir/stdout" 'seed=18446744073709551615 child=unset arg1=two words arg2=*'

empty_seed_dir="$test_tmp/empty-seed"
gomad_run_checked 5 0 'exec wrapper seed=empty mode=transfer iteration=0' "$empty_seed_dir" -- \
	env GOMADV3_CHILD_SEED= "$exec_wrapper" bash -c 'printf "<%s>\n" "$GOMADSEED"'
require_file "$empty_seed_dir/stdout" '<>'

malformed_seed_dir="$test_tmp/malformed-seed"
gomad_run_checked 5 0 'exec wrapper seed=malformed mode=transfer iteration=0' "$malformed_seed_dir" -- \
	env GOMADV3_CHILD_SEED=not-a-seed "$exec_wrapper" bash -c 'printf "%s\n" "$GOMADSEED"'
require_file "$malformed_seed_dir/stdout" 'not-a-seed'

target_status_dir="$test_tmp/target-status"
gomad_run_checked 5 37 'exec wrapper seed=0 mode=target-status iteration=0' "$target_status_dir" -- \
	env GOMADV3_CHILD_SEED=0 "$exec_wrapper" bash -c 'exit 37'

target_signal_dir="$test_tmp/target-signal"
gomad_run_checked 5 143 'exec wrapper seed=1 mode=target-signal iteration=0' "$target_signal_dir" -- \
	env GOMADV3_CHILD_SEED=1 "$exec_wrapper" bash -c 'kill -TERM $$'

malformed_patch="$test_tmp/malformed.patch"
printf '%s\n' \
	'diff --git a/src/runtime/proc.go b/src/runtime/proc.go' \
	'--- a/src/runtime/proc.go' \
	'+++ b/src/runtime/proc.go' \
	'@@ -1 +1 @@' \
	'this context line has no unified-diff prefix' >"$malformed_patch"
if GOMADV3_PATCH_FILE="$malformed_patch" "$script_dir/test.sh" validate \
	>"$test_tmp/malformed-patch-stdout" 2>"$test_tmp/malformed-patch-stderr"; then
	fail 'gomadv3 patch validation accepted malformed unified-diff syntax'
fi
if ! grep -Fq 'gomadv3 patch is malformed' "$test_tmp/malformed-patch-stderr"; then
	fail 'gomadv3 malformed-patch diagnostic is absent'
fi

wrong_version_root="$test_tmp/wrong-version-go"
mkdir -p "$wrong_version_root"
printf 'go1.26.3\n' >"$wrong_version_root/VERSION"
if "$script_dir/regenerate-patch.sh" "$wrong_version_root" \
	>"$test_tmp/regenerate-wrong-version-stdout" 2>"$test_tmp/regenerate-wrong-version-stderr"; then
	fail 'gomadv3 patch regeneration accepted the wrong Go version'
fi
if ! grep -Fq 'gomadv3 patch candidate must be go1.26.4' "$test_tmp/regenerate-wrong-version-stderr"; then
	fail 'gomadv3 patch regeneration wrong-version diagnostic is absent'
fi

printf 'gomadv3 checked runner tests passed\n'
