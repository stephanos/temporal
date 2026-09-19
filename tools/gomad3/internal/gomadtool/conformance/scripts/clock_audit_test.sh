#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
root=$(cd "$script_dir/../../../.." && pwd)
go_bin="$root/.toolchain/bin/go"
probe="$script_dir/clock_audit.d"
test_tmp=$(mktemp -d "$root/.toolchain/clock-audit.XXXXXX")
cleanup() {
	find "$test_tmp" -depth -delete
}
trap cleanup EXIT

env -u GOMADSEED -u GOMAD3_CHILD_SEED CGO_ENABLED=0 GOWORK=off \
	"$go_bin" -C "$root/internal/gomadtool/conformance/testdata" build -trimpath -o "$test_tmp/clock-audit" ./clock_audit
"$go_bin" tool nm "$test_tmp/clock-audit" >"$test_tmp/clock-audit.nm"
if ! grep -Fq ' main.auditStart' "$test_tmp/clock-audit.nm"; then
	printf 'gomad3 clock audit marker is absent from the probe binary\n' >&2
	exit 1
fi

if [[ $EUID -eq 0 ]]; then
	dtrace_sudo=()
else
	if ! sudo -n true 2>/dev/null; then
		printf 'gomad3 clock audit requires root DTrace privileges\n' >&2
		exit 1
	fi
	dtrace_sudo=(sudo -n)
fi

run_trace() {
	local name=$1
	local seed=$2
	local output="$test_tmp/$name.trace"
	local -a environment=(/usr/bin/env -u GOMADSEED -u GOMAD3_CHILD_SEED TZ=UTC)
	if [[ -n $seed ]]; then
		environment+=(GOMADSEED="$seed")
	fi
	if ! "${dtrace_sudo[@]}" "${environment[@]}" /usr/sbin/dtrace -q -s "$probe" -c "$test_tmp/clock-audit" >"$output" 2>&1; then
		printf 'gomad3 clock audit tracer failed for %s\n' "$name" >&2
		cat "$output" >&2
		exit 1
	fi
	if grep -Eq 'dtrace: (error|failed)|DTrace requires additional privileges' "$output"; then
		printf 'gomad3 clock audit tracer was unavailable for %s\n' "$name" >&2
		cat "$output" >&2
		exit 1
	fi
	if ! grep -Fq 'GOMAD3_AUDIT_START' "$output"; then
		printf 'gomad3 clock audit did not observe its activation marker for %s\n' "$name" >&2
		cat "$output" >&2
		exit 1
	fi
}

run_trace positive ''
for function in clock_gettime mach_absolute_time; do
	if ! grep -Eq "GOMAD3_HOST_CLOCK .*${function}" "$test_tmp/positive.trace"; then
		printf 'gomad3 clock audit positive control missed %s\n' "$function" >&2
		cat "$test_tmp/positive.trace" >&2
		exit 1
	fi
done

run_trace deterministic 1
if grep -Fq 'GOMAD3_HOST_CLOCK ' "$test_tmp/deterministic.trace"; then
	printf 'gomad3 deterministic target reached an ambient host clock\n' >&2
	cat "$test_tmp/deterministic.trace" >&2
	exit 1
fi

printf 'gomad3 host-clock escape audit passed\n'
