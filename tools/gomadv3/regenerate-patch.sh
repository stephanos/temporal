#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$script_dir/toolchain-version.sh"

if [[ $# -ne 1 ]]; then
	printf 'usage: %s GO-SOURCE-ROOT\n' "$0" >&2
	exit 1
fi

candidate_root=$1
candidate_version=
if [[ -f "$candidate_root/VERSION" ]]; then
	IFS= read -r candidate_version <"$candidate_root/VERSION" || true
fi
if [[ ! -d "$candidate_root" || "$candidate_version" != "$go_version" ]]; then
	printf 'gomadv3 patch candidate must be %s\n' "$go_version" >&2
	exit 1
fi

candidate_root=$(cd "$candidate_root" && pwd -P)
patch_output=${GOMADV3_PATCH_OUTPUT:-"$script_dir/go1.26.4.patch"}
toolchain_dir="$script_dir/.toolchain"
archive_dir="$toolchain_dir/downloads"
archive_path="$archive_dir/$archive_name"
work_dir=
download_tmp=
patch_tmp=

cleanup() {
	local status=$?
	trap - EXIT
	if [[ -n "$work_dir" ]]; then
		rm -rf "$work_dir"
	fi
	if [[ -n "$download_tmp" ]]; then
		rm -f "$download_tmp"
	fi
	if [[ -n "$patch_tmp" ]]; then
		rm -f "$patch_tmp"
	fi
	exit "$status"
}
trap cleanup EXIT

sha256_file() {
	shasum -a 256 "$1" | awk '{print $1}'
}

bootstrap_go=${GOMADV3_BOOTSTRAP_GO:-}
if [[ -z "$bootstrap_go" ]]; then
	bootstrap_go=$(command -v go || true)
fi
if [[ -z "$bootstrap_go" || ! -x "$bootstrap_go" ]]; then
	printf 'gomadv3 requires an installed bootstrap Go; set GOMADV3_BOOTSTRAP_GO\n' >&2
	exit 1
fi
bootstrap_root=$(env -u GOMADSEED "$bootstrap_go" env GOROOT)
gofmt_bin="$bootstrap_root/bin/gofmt"
if [[ ! -x "$gofmt_bin" ]]; then
	printf 'gomadv3 bootstrap gofmt is missing: %s\n' "$gofmt_bin" >&2
	exit 1
fi

special=
while IFS= read -r -d '' special; do
	break
done < <(find "$candidate_root" -mindepth 1 ! -type d ! -type f -print0)
if [[ -n "$special" ]]; then
	printf 'gomadv3 patch candidate contains a non-regular entry: %s\n' "$special" >&2
	exit 1
fi

mkdir -p "$archive_dir" "$(dirname "$patch_output")"
if [[ ! -f "$archive_path" || $(sha256_file "$archive_path") != "$archive_sha256" ]]; then
	download_tmp=$(mktemp "$archive_dir/$archive_name.XXXXXX")
	curl --fail --location --retry 3 --output "$download_tmp" "$archive_url"
	if [[ $(sha256_file "$download_tmp") != "$archive_sha256" ]]; then
		printf 'checksum mismatch for %s\n' "$archive_url" >&2
		exit 1
	fi
	mv "$download_tmp" "$archive_path"
	download_tmp=
fi

work_dir=$(mktemp -d "$toolchain_dir/regenerate-patch.XXXXXX")
tar -C "$work_dir" -xzf "$archive_path"
pristine_root="$work_dir/go"
changes_file="$work_dir/changes"
diff_status=0
git diff --no-index --name-status -z --no-renames "$pristine_root" "$candidate_root" >"$changes_file" || diff_status=$?
if ((diff_status > 1)); then
	printf 'gomadv3 patch candidate comparison failed\n' >&2
	exit 1
fi
changed=()
while IFS= read -r -d '' status && IFS= read -r -d '' path; do
	case "$status" in
	A)
		relative=${path#"$candidate_root"/}
		printf 'gomadv3 patch candidate adds a source path: %s\n' "$relative" >&2
		exit 1
		;;
	D)
		relative=${path#"$pristine_root"/}
		printf 'gomadv3 patch candidate deletes a source path: %s\n' "$relative" >&2
		exit 1
		;;
	M)
		relative=${path#"$pristine_root"/}
		changed+=("$relative")
		;;
	*)
		printf 'gomadv3 patch candidate has unsupported change %s: %s\n' "$status" "$path" >&2
		exit 1
		;;
	esac
done <"$changes_file"
if ((${#changed[@]} == 0)); then
	printf 'gomadv3 patch candidate contains no changes\n' >&2
	exit 1
fi

git -C "$pristine_root" init -q
git -C "$pristine_root" config core.autocrlf false
git -C "$pristine_root" config core.filemode true
git -C "$pristine_root" add -- "${changed[@]}"
for relative in "${changed[@]}"; do
	cp "$candidate_root/$relative" "$pristine_root/$relative"
	if [[ "$relative" == *.go ]]; then
		"$gofmt_bin" -w "$pristine_root/$relative"
	fi
done

patch_tmp=$(mktemp "$(dirname "$patch_output")/.gomadv3-patch.XXXXXX")
LC_ALL=C git -C "$pristine_root" diff --no-ext-diff --binary \
	--src-prefix=a/ --dst-prefix=b/ -- >"$patch_tmp"
if [[ ! -s "$patch_tmp" ]]; then
	printf 'gomadv3 patch regeneration produced no changes\n' >&2
	exit 1
fi
GOMADV3_PATCH_FILE="$patch_tmp" "$script_dir/test.sh" validate
git -C "$pristine_root" apply --cached --check "$patch_tmp"
chmod 0644 "$patch_tmp"
mv "$patch_tmp" "$patch_output"
patch_tmp=
