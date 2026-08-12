#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$script_dir/toolchain-version.sh"
toolchain_dir="$script_dir/.toolchain"
patch_file=${GOMADV3_PATCH_FILE:-"$script_dir/go1.26.4.patch"}
overlay_dir=${GOMADV3_OVERLAY_DIR:-"$script_dir/overlay"}
patch_snapshot=
overlay_snapshot=
download_tmp=
work_dir=
lock_path=
lock_owner_file=
owns_lock=false
build_key=
build_environment=canonical-v4
build_path=/usr/bin:/bin:/usr/sbin:/sbin:/usr/xpg4/bin:/opt/freeware/bin:/usr/local/bin:/opt/homebrew/bin:/opt/local/bin
build_bash=$BASH
build_bash_version=$BASH_VERSION

cleanup() {
	local status=$?
	trap - EXIT
	if [[ -n "$download_tmp" ]]; then
		rm -f "$download_tmp"
	fi
	if [[ -n "$work_dir" ]]; then
		rm -rf "$work_dir"
	fi
	if [[ -n "$patch_snapshot" ]]; then
		rm -f "$patch_snapshot"
	fi
	if [[ -n "$overlay_snapshot" ]]; then
		rm -rf "$overlay_snapshot"
	fi
	if [[ "$owns_lock" == true ]]; then
		rm -f "$lock_owner_file"
		rmdir "$lock_path" 2>/dev/null || true
	elif [[ -n "$lock_owner_file" ]]; then
		rm -f "$lock_owner_file"
		rmdir "$lock_path" 2>/dev/null || true
	fi
	if [[ $status -ne 0 && -n "$build_key" ]]; then
		printf 'gomadv3 toolchain build failed (key %s)\n' "$build_key" >&2
	fi
	exit "$status"
}

trap cleanup EXIT

sha256_file() {
	shasum -a 256 "$1" | awk '{print $1}'
}

sorted_files() {
	local root=$1
	local LC_ALL=C
	local files=() file swap
	local i j
	while IFS= read -r -d '' file; do
		files+=("$file")
	done < <(find "$root" -type f -print0)
	for ((i = 0; i < ${#files[@]}; i++)); do
		for ((j = i + 1; j < ${#files[@]}; j++)); do
			if [[ ${files[j]} < ${files[i]} ]]; then
				swap=${files[i]}
				files[i]=${files[j]}
				files[j]=$swap
			fi
		done
	done
	if ((${#files[@]} > 0)); then
		printf '%s\0' "${files[@]}"
	fi
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
bootstrap_version=$(env -u GOMADSEED "$bootstrap_go" version)
host_os=$(env -u GOMADSEED "$bootstrap_go" env GOHOSTOS)
host_arch=$(env -u GOMADSEED "$bootstrap_go" env GOHOSTARCH)
case "$host_os" in
	aix | darwin | dragonfly | freebsd | illumos | linux | netbsd | openbsd | solaris) ;;
	*)
		printf 'gomadv3 deterministic mode does not support host OS %s\n' "$host_os" >&2
		exit 1
		;;
esac

if [[ ! -s "$patch_file" ]]; then
	printf 'gomadv3 patch is missing: %s\n' "$patch_file" >&2
	exit 1
fi

mkdir -p "$toolchain_dir"
GOMADV3_PATCH_FILE="$patch_file" GOMADV3_OVERLAY_DIR="$overlay_dir" "$script_dir/test.sh" validate
patch_snapshot=$(mktemp "$toolchain_dir/patch.XXXXXX")
cp "$patch_file" "$patch_snapshot"
overlay_snapshot=$(mktemp -d "$toolchain_dir/overlay.XXXXXX")
cp -R "$overlay_dir/." "$overlay_snapshot/"
GOMADV3_PATCH_FILE="$patch_snapshot" GOMADV3_OVERLAY_DIR="$overlay_snapshot" "$script_dir/test.sh" validate

patch_sha256=$(sha256_file "$patch_snapshot")
overlay_sha256=$(
	(
		cd "$overlay_snapshot"
		while IFS= read -r -d '' path; do
			printf '%s\0%s\0' "${path#./}" "$(sha256_file "$path")"
		done < <(sorted_files .)
	) | shasum -a 256 | awk '{print $1}'
)
build_key=$(printf '%s\n' "$go_version" "$archive_sha256" "$patch_sha256" "$overlay_sha256" "$host_os" "$host_arch" "$bootstrap_version" "$build_environment" "$build_path" "$build_bash" "$build_bash_version" | shasum -a 256 | awk '{print $1}')
build_dir="$toolchain_dir/builds/$build_key"
archive_dir="$toolchain_dir/downloads"
archive_path="$archive_dir/$archive_name"
stamp_path="$toolchain_dir/build-key"
lock_root="$toolchain_dir/locks"
lock_path="$lock_root/$build_key"
lock_owner_name="owner.$$.${RANDOM}.${RANDOM}"

mkdir -p "$archive_dir" "$toolchain_dir/builds" "$lock_root"

lock_has_only_owner() {
	local expected=$1
	local child
	local count=0
	while IFS= read -r -d '' child; do
		count=$((count + 1))
		if [[ "$child" != "$expected" ]]; then
			return 1
		fi
	done < <(find "$lock_path" -mindepth 1 -maxdepth 1 -type f -print0 2>/dev/null)
	((count == 1))
}

acquire_build_lock() {
	local attempt observed owner stale_path
	for ((attempt = 1; attempt <= 600; attempt++)); do
		if mkdir "$lock_path" 2>/dev/null; then
			lock_owner_file="$lock_path/$lock_owner_name"
			if printf '%s\n' "$$" >"$lock_owner_file" 2>/dev/null && lock_has_only_owner "$lock_owner_file"; then
				owns_lock=true
				return
			fi
			rm -f "$lock_owner_file"
			lock_owner_file=
			rmdir "$lock_path" 2>/dev/null || true
			continue
		fi
		observed=
		while IFS= read -r -d '' observed; do
			break
		done < <(find "$lock_path" -mindepth 1 -maxdepth 1 -type f -name 'owner.*' -print0 2>/dev/null)
		if [[ -z "$observed" ]]; then
			rmdir "$lock_path" 2>/dev/null || true
			continue
		fi
		owner=$(cat "$observed" 2>/dev/null || true)
		if [[ "$owner" =~ ^[0-9]+$ ]] && ! kill -0 "$owner" 2>/dev/null; then
			stale_path="$lock_root/$build_key.stale.$$.${RANDOM}"
			if mv "$observed" "$stale_path" 2>/dev/null; then
				if rmdir "$lock_path" 2>/dev/null; then
					rm -f "$stale_path"
				else
					mv "$stale_path" "$observed" 2>/dev/null || true
				fi
			fi
			continue
		fi
		if ((attempt == 1)); then
			printf 'waiting for gomadv3 build key %s\n' "$build_key"
		fi
		sleep 1
	done
	printf 'timed out waiting for gomadv3 build key %s\n' "$build_key" >&2
	exit 1
}

build_complete() {
	[[ -x "$build_dir/bin/go" ]] && [[ $(env -u GOMADSEED "$build_dir/bin/go" version) == *" $go_version "* ]]
}

acquire_build_lock

publish_toolchain() {
	local bin_dir="$toolchain_dir/bin"
	local go_tmp="$bin_dir/go.next.$$"
	local stamp_tmp="$stamp_path.next.$$"
	if [[ -L "$bin_dir" ]]; then
		unlink "$bin_dir"
	fi
	mkdir -p "$bin_dir"
	printf '%s\n' \
		'#!/bin/sh' \
		'toolchain_dir=$(CDPATH= cd "$(dirname "$0")/.." && pwd) || exit' \
		'build_key=$(cat "$toolchain_dir/build-key") || exit' \
		'exec "$toolchain_dir/builds/$build_key/bin/go" "$@"' >"$go_tmp"
	chmod +x "$go_tmp"
	printf '%s\n' "$build_key" >"$stamp_tmp"
	mv -f "$stamp_tmp" "$stamp_path"
	mv -f "$go_tmp" "$bin_dir/go"
}

if build_complete; then
	publish_toolchain
	printf 'gomadv3 toolchain is ready (%s/%s, key %s)\n' "$host_os" "$host_arch" "$build_key"
	exit 0
fi

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

work_dir=$(mktemp -d "$toolchain_dir/build.XXXXXX")
mkdir -p "$work_dir/bootstrap-cache" "$work_dir/tmp"
tar -C "$work_dir" -xzf "$archive_path"
while IFS= read -r -d '' source; do
	relative=${source#"$overlay_snapshot"/}
	destination="$work_dir/go/$relative"
	if [[ -e "$destination" || -L "$destination" ]]; then
		printf 'gomadv3 overlay collides with upstream Go source: %s\n' "$relative" >&2
		exit 1
	fi
done < <(sorted_files "$overlay_snapshot")
"$script_dir/materialize-patch.sh" "$work_dir/go" "$patch_snapshot"
while IFS= read -r -d '' source; do
	relative=${source#"$overlay_snapshot"/}
	destination="$work_dir/go/$relative"
	mkdir -p "$(dirname "$destination")"
	cp "$source" "$destination"
done < <(sorted_files "$overlay_snapshot")
(
	cd "$work_dir/go/src"
	env -i \
		BOOT_GO_GCFLAGS= \
		BOOT_GO_LDFLAGS= \
		CC= \
		CC_FOR_TARGET= \
		CGO_ENABLED=0 \
		CXX= \
		CXX_FOR_TARGET= \
		FC= \
		GOBUILDTIMELOGFILE= \
		GODEBUG= \
		GOCACHE="$work_dir/bootstrap-cache" \
		GO386= \
		GOAMD64= \
		GOARCH="$host_arch" \
		GOARM= \
		GOARM64= \
		GOBOOTSTRAP_TOOLEXEC= \
		GO_BUILDER_NAME= \
		GO_DISTFLAGS= \
		GOENV=off \
		GOEXPERIMENT= \
		GO_EXTLINK_ENABLED= \
		GO_GCFLAGS= \
		GO_LDFLAGS= \
		GO_LDSO= \
		GOFIPS140= \
		GOFLAGS= \
		GOHOSTARCH="$host_arch" \
		GOHOSTOS="$host_os" \
		GOMIPS= \
		GOMIPS64= \
		GOOS="$host_os" \
		GOPPC64= \
		GORISCV64= \
		GOTOOLCHAIN=local \
		GOWORK=off \
		GOWASM= \
		GOROOT_BOOTSTRAP="$bootstrap_root" \
		LC_ALL=C \
		PATH="$build_path" \
		PKG_CONFIG= \
		TMPDIR="$work_dir/tmp" \
		TZ=UTC \
		"$build_bash" ./make.bash
)
if [[ $(env -u GOMADSEED "$work_dir/go/bin/go" version) != *" $go_version "* ]]; then
	printf 'built toolchain reported an unexpected version\n' >&2
	exit 1
fi

if build_complete; then
	publish_toolchain
	printf 'gomadv3 toolchain is ready (%s/%s, key %s)\n' "$host_os" "$host_arch" "$build_key"
	exit 0
fi
if [[ -e "$build_dir" ]]; then
	incomplete_dir="$build_dir.incomplete.$$"
	mv "$build_dir" "$incomplete_dir"
	rm -rf "$incomplete_dir"
fi
mv "$work_dir/go" "$build_dir"
rm -rf "$work_dir"
work_dir=

publish_toolchain
printf 'gomadv3 toolchain is ready (%s/%s, key %s)\n' "$host_os" "$host_arch" "$build_key"
