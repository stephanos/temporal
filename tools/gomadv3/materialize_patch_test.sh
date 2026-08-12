#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
test_tmp=$(mktemp -d)
cleanup() {
	rm -rf "$test_tmp"
}
trap cleanup EXIT

make_source() {
	local root=$1
	mkdir -p "$root/src/runtime"
	printf '%s\n' alpha context target tail omega >"$root/src/runtime/proc.go"
}

exact_patch="$test_tmp/exact.patch"
printf '%s\n' \
	'diff --git a/src/runtime/proc.go b/src/runtime/proc.go' \
	'--- a/src/runtime/proc.go' \
	'+++ b/src/runtime/proc.go' \
	'@@ -1,5 +1,5 @@' \
	' alpha' \
	' context' \
	'-target' \
	'+replacement' \
	' tail' \
	' omega' >"$exact_patch"
exact_root="$test_tmp/exact"
make_source "$exact_root"
"$script_dir/materialize-patch.sh" "$exact_root" "$exact_patch"
if [[ $(<"$exact_root/src/runtime/proc.go") != $'alpha\ncontext\nreplacement\ntail\nomega' ]]; then
	printf 'gomadv3 exact patch was not materialized\n' >&2
	exit 1
fi
if [[ -e "$exact_root/src/runtime/proc.go.orig" ]]; then
	printf 'gomadv3 exact patch left a backup file\n' >&2
	exit 1
fi

relative_root="$test_tmp/relative"
make_source "$relative_root"
(
	cd "$test_tmp"
	"$script_dir/materialize-patch.sh" relative exact.patch
)
if [[ $(<"$relative_root/src/runtime/proc.go") != $'alpha\ncontext\nreplacement\ntail\nomega' ]]; then
	printf 'gomadv3 relative patch path was not materialized\n' >&2
	exit 1
fi

nonapplying_patch="$test_tmp/nonapplying.patch"
sed 's/-target/-missing/' "$exact_patch" >"$nonapplying_patch"
nonapplying_root="$test_tmp/nonapplying"
make_source "$nonapplying_root"
if "$script_dir/materialize-patch.sh" "$nonapplying_root" "$nonapplying_patch" \
	>"$test_tmp/nonapplying-stdout" 2>"$test_tmp/nonapplying-stderr"; then
	printf 'gomadv3 materialization accepted a nonapplying patch\n' >&2
	exit 1
fi
if ! grep -Fq 'gomadv3 patch does not apply with zero fuzz' "$test_tmp/nonapplying-stderr"; then
	printf 'gomadv3 nonapplying-patch diagnostic is absent\n' >&2
	exit 1
fi
if [[ $(<"$nonapplying_root/src/runtime/proc.go") != $'alpha\ncontext\ntarget\ntail\nomega' ]]; then
	printf 'gomadv3 failed patch mutated its source tree\n' >&2
	exit 1
fi

fuzzy_patch="$test_tmp/fuzzy.patch"
sed 's/^ alpha$/ absent/' "$exact_patch" >"$fuzzy_patch"
fuzzy_root="$test_tmp/fuzzy"
make_source "$fuzzy_root"
if ! (
	cd "$fuzzy_root"
	patch --dry-run --batch -V never -p1 -F 1 <"$fuzzy_patch" >/dev/null
); then
	printf 'gomadv3 fuzz fixture does not apply with fuzz enabled\n' >&2
	exit 1
fi
if "$script_dir/materialize-patch.sh" "$fuzzy_root" "$fuzzy_patch" \
	>"$test_tmp/fuzzy-stdout" 2>"$test_tmp/fuzzy-stderr"; then
	printf 'gomadv3 materialization accepted a fuzz-dependent patch\n' >&2
	exit 1
fi
if [[ $(<"$fuzzy_root/src/runtime/proc.go") != $'alpha\ncontext\ntarget\ntail\nomega' ]]; then
	printf 'gomadv3 fuzz-dependent patch mutated its source tree\n' >&2
	exit 1
fi

printf 'gomadv3 patch materialization tests passed\n'
