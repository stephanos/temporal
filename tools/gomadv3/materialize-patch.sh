#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 2 || ! -d "$1" || ! -s "$2" ]]; then
	printf 'usage: %s GO-SOURCE-ROOT PATCH-FILE\n' "$0" >&2
	exit 1
fi

source_root=$1
patch_file=$2
patch_file=$(cd "$(dirname "$patch_file")" && pwd -P)/$(basename "$patch_file")
if ! (
	cd "$source_root"
	patch --dry-run --batch -V none -p1 -F 0 <"$patch_file"
); then
	printf 'gomadv3 patch does not apply with zero fuzz\n' >&2
	exit 1
fi
(
	cd "$source_root"
	patch --batch -V none -p1 -F 0 <"$patch_file"
)
