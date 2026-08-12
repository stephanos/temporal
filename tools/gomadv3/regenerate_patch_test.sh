#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$script_dir/toolchain-version.sh"

test_tmp=$(mktemp -d)
cleanup() {
	rm -rf "$test_tmp"
}
trap cleanup EXIT

archive_path="$script_dir/.toolchain/downloads/$archive_name"
if [[ ! -f "$archive_path" ]]; then
	printf 'gomadv3 patch regeneration test requires %s\n' "$archive_path" >&2
	exit 1
fi

mkdir -p "$test_tmp/candidate" "$test_tmp/pristine"
tar -C "$test_tmp/candidate" -xzf "$archive_path"
tar -C "$test_tmp/pristine" -xzf "$archive_path"
candidate_root="$test_tmp/candidate/go"
printf '// gomadv3 patch regeneration fixture\n' >>"$candidate_root/src/runtime/rand.go"
gofmt -w "$candidate_root/src/runtime/rand.go"

first_patch="$test_tmp/first.patch"
second_patch="$test_tmp/second.patch"
GOMADV3_PATCH_OUTPUT="$first_patch" "$script_dir/regenerate-patch.sh" "$candidate_root"
GOMADV3_PATCH_OUTPUT="$second_patch" "$script_dir/regenerate-patch.sh" "$candidate_root"
if ! cmp -s "$first_patch" "$second_patch"; then
	printf 'gomadv3 patch regeneration is not deterministic\n' >&2
	exit 1
fi
numstat=$(cd "$test_tmp/pristine/go" && git apply --numstat <"$first_patch")
if [[ "$numstat" != $'2\t0\tsrc/runtime/rand.go' ]]; then
	printf 'gomadv3 patch regeneration emitted an unexpected change set: %q\n' "$numstat" >&2
	sed -n '1,80p' "$first_patch" >&2
	exit 1
fi
(
	cd "$test_tmp/pristine/go"
	patch --dry-run --batch -p1 -F 0 <"$first_patch"
	patch --batch -p1 -F 0 <"$first_patch"
)
if ! cmp -s "$candidate_root/src/runtime/rand.go" "$test_tmp/pristine/go/src/runtime/rand.go"; then
	printf 'gomadv3 regenerated patch did not reproduce the candidate source\n' >&2
	exit 1
fi

printf 'gomadv3 patch regeneration tests passed\n'
