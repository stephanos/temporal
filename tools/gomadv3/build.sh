#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "$script_dir/toolchain-version.sh"
toolchain_dir=${GOMADV3_TOOLCHAIN_DIR:-"$script_dir/.toolchain"}
if [[ "$toolchain_dir" != /* || "$toolchain_dir" == / ]]; then
	printf 'gomadv3 toolchain directory must be an absolute non-root path: %s\n' "$toolchain_dir" >&2
	exit 1
fi
if [[ -n ${GOMADV3_TEST_FAIL_PHASE:-} && ${GOMADV3_TESTING:-} != 1 ]]; then
	printf 'gomadv3 builder failure injection requires GOMADV3_TESTING=1\n' >&2
	exit 1
fi
patch_file=${GOMADV3_PATCH_FILE:-"$script_dir/$patch_name"}
overlay_dir=${GOMADV3_OVERLAY_DIR:-"$script_dir/overlay"}
patch_snapshot=
overlay_snapshot=
download_tmp=
work_dir=
launcher_tmp=
stamp_tmp=
lock_path=
lock_owner_file=
owns_lock=false
build_key=
build_environment=canonical-v5
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
	if [[ -n "$launcher_tmp" ]]; then
		rm -f "$launcher_tmp"
	fi
	if [[ -n "$stamp_tmp" ]]; then
		rm -f "$stamp_tmp"
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

fail_at() {
	if [[ ${GOMADV3_TESTING:-} == 1 && ${GOMADV3_TEST_FAIL_PHASE:-} == "$1" ]]; then
		printf 'gomadv3 injected builder failure: %s\n' "$1" >&2
		exit 86
	fi
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
host_platform="$host_os/$host_arch"
host_supported=false
for qualified_platform in "${qualified_platforms[@]}"; do
	if [[ "$host_platform" == "$qualified_platform" ]]; then
		host_supported=true
		break
	fi
done
if [[ "$host_supported" == false ]]; then
	printf 'gomadv3 complete deterministic mode requires host %s; got %s\n' \
		"$(IFS=,; printf '%s' "${qualified_platforms[*]}")" "$host_platform" >&2
	exit 1
fi

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

build_key=$(
	env -u GOMADSEED GOCACHE="$toolchain_dir/generator-cache" GOWORK=off \
		"$bootstrap_go" -C "$script_dir" run ./internal/hosttool build-key \
		--go-version="$go_version" \
		--archive-sha256="$archive_sha256" \
		--patch="$patch_snapshot" \
		--overlay="$overlay_snapshot" \
		--host-os="$host_os" \
		--host-arch="$host_arch" \
		--bootstrap-version="$bootstrap_version" \
		--recipe-version="$build_environment" \
		--build-path="$build_path" \
		--bash-path="$build_bash" \
		--bash-version="$build_bash_version"
)
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
fail_at after-lock

publish_toolchain() {
	local bin_dir="$toolchain_dir/bin"
	launcher_tmp="$bin_dir/go.next.$$"
	stamp_tmp="$stamp_path.next.$$"
	if [[ -L "$bin_dir" ]]; then
		unlink "$bin_dir"
	fi
	mkdir -p "$bin_dir"
	printf '%s\n' \
		'#!/bin/sh' \
		'toolchain_dir=$(CDPATH= cd "$(dirname "$0")/.." && pwd) || exit' \
		'build_key=$(cat "$toolchain_dir/build-key") || exit' \
		'unset GOROOT' \
		'exec "$toolchain_dir/builds/$build_key/bin/go" "$@"' >"$launcher_tmp"
	chmod +x "$launcher_tmp"
	printf '%s\n' "$build_key" >"$stamp_tmp"
	mv -f "$stamp_tmp" "$stamp_path"
	stamp_tmp=
	fail_at after-stamp-publish
	mv -f "$launcher_tmp" "$bin_dir/go"
	launcher_tmp=
	fail_at after-launcher-publish
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
fail_at after-extract
while IFS= read -r -d '' source; do
	relative=${source#"$overlay_snapshot"/}
	destination="$work_dir/go/$relative"
	if [[ -e "$destination" || -L "$destination" ]]; then
		printf 'gomadv3 overlay collides with upstream Go source: %s\n' "$relative" >&2
		exit 1
	fi
done < <(sorted_files "$overlay_snapshot")
"$script_dir/materialize-patch.sh" "$work_dir/go" "$patch_snapshot"
fail_at after-patch
while IFS= read -r -d '' source; do
	relative=${source#"$overlay_snapshot"/}
	destination="$work_dir/go/$relative"
	mkdir -p "$(dirname "$destination")"
	cp "$source" "$destination"
done < <(sorted_files "$overlay_snapshot")
fail_at after-overlay
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
fail_at after-compile

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
fail_at after-build-publish

publish_toolchain
printf 'gomadv3 toolchain is ready (%s/%s, key %s)\n' "$host_os" "$host_arch" "$build_key"
