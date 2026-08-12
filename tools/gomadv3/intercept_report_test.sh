#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
go_bin="$script_dir/.toolchain/bin/go"
expected="$script_dir/expected-intercepts-go1.26.4.txt"
test_tmp=$(mktemp -d "$script_dir/.toolchain/intercept-report.XXXXXX")
cleanup() {
	find "$test_tmp" -depth -delete
}
trap cleanup EXIT

for package in os net; do
	env -u GOMADSEED -u GOMADV3_CHILD_SEED GOCACHE="$test_tmp/cache-$package" GOWORK=off \
		"$go_bin" install -a -gcflags="$package=-m" "$package" \
		>"$test_tmp/$package.stdout" 2>"$test_tmp/$package.stderr"
	awk '/^gomad intercept applied: / { sub(/^gomad intercept applied: /, ""); print }' \
		"$test_tmp/$package.stderr" >>"$test_tmp/actual"
done

LC_ALL=C sort -o "$test_tmp/actual" "$test_tmp/actual"
if ! cmp -s "$expected" "$test_tmp/actual"; then
	printf 'gomadv3 applied interceptions differ from the Go 1.26.4 manifest\n' >&2
	diff -u "$expected" "$test_tmp/actual" >&2 || true
	exit 1
fi

printf 'gomadv3 interception report tests passed\n'
