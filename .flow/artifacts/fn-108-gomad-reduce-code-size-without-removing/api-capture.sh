#!/bin/sh
# fn-108 public-surface capture for R8: `go doc -all` of every public Gomad
# package and the usage/--help output, stream routing and exit status of every
# gomad and gomadtool command.
#
# usage: api-capture.sh OUTDIR BINDIR
#   OUTDIR  absolute path that does not exist yet; receives the capture
#   BINDIR  absolute path outside the repository that does not exist yet;
#           receives the two built binaries and two reused temporary files
# The script creates both directories, writes its own files only beneath them
# and never deletes anything. The Go commands it runs use the ambient Go build
# and module caches like any other build. It needs the patched toolchain at
# tools/gomad3/.toolchain and fails when a package fails to load or document.
set -eu
LC_ALL=C
export LC_ALL

fail() {
	echo "api-capture: $1" >&2
	exit 2
}

[ "$#" -eq 2 ] || fail "usage: api-capture.sh OUTDIR BINDIR"
out=$1
bin=$2
case "$out" in /*) ;; *) fail "OUTDIR must be absolute" ;; esac
case "$bin" in /*) ;; *) fail "BINDIR must be absolute" ;; esac

# Resolves the parent directory physically, so `..` and symbolic links cannot
# place a target somewhere other than where its spelling suggests.
canonical() {
	parent=$(cd "$(dirname "$1")" 2> /dev/null && pwd -P) || fail "parent directory does not exist: $1"
	leaf=$(basename "$1")
	case "$leaf" in . | .. | /) fail "unusable path: $1" ;; esac
	printf '%s/%s\n' "${parent%/}" "$leaf"
}

out=$(canonical "$out")
bin=$(canonical "$bin")
[ ! -e "$out" ] && [ ! -L "$out" ] || fail "OUTDIR already exists: $out"
[ ! -e "$bin" ] && [ ! -L "$bin" ] || fail "BINDIR already exists: $bin"

top=$(cd "$(git rev-parse --show-toplevel)" && pwd -P)
case "$bin/" in "$top"/*) fail "BINDIR must be outside the repository" ;; esac
module="$top/tools/gomad3"
tgo="$module/.toolchain/bin/go"
[ -x "$tgo" ] || fail "missing patched toolchain: $tgo (run make -C tools/gomad3 toolchain)"

mkdir -p "$out/go-doc" "$out/cli" "$bin"

# One pinned environment for every Go invocation: no persistent `go env -w`
# settings, no ambient target platform or experiments, no workspace, no
# toolchain switch, and no module-file writes.
unset GOMADSEED GOMAD3_CHILD_SEED GOOS GOARCH GOEXPERIMENT GOARM64 GOAMD64
GOENV=off
GOFLAGS=-mod=readonly
GOWORK=off
GOTOOLCHAIN=local
CGO_ENABLED=0
TZ=UTC
export GOENV GOFLAGS GOWORK GOTOOLCHAIN CGO_ENABLED TZ

# Public Go surface: every package of the nested module outside internal/, cmd/
# and the runtime overlay that has non-test Go files, plus the gomad3sim sibling
# package in the root module. The module root package holds only
# architecture_test.go and has no importable surface.
cd "$module"
prefix=go.temporal.io/server/tools/gomad3
# `go list ./...` without -e fails on the runtime overlay packages, which import
# packages internal to the Go tree. -e keeps the listing going, and the template
# marks a package that failed to load so a public one is not silently skipped.
listed=$("$tgo" list -e \
	-f '{{if .GoFiles}}{{.ImportPath}}{{if or .Error .DepsErrors}} LOAD-ERROR{{end}}{{end}}' ./... 2> /dev/null)
public=$(printf '%s\n' "$listed" |
	grep -v -e '/internal/' -e '/internal$' -e '/internal ' -e '/cmd/' -e '/toolchain/runtime/overlay/' |
	sort)
[ -n "$public" ] || fail "no public packages listed"
case "$public" in *LOAD-ERROR*) fail "a public package failed to load: $public" ;; esac
printf '%s\n' "$public" > "$out/go-doc/packages.txt"
while IFS= read -r package; do
	name=$(printf '%s\n' "${package#"$prefix"}" | sed -e 's|^/||' -e 's|/|_|g')
	[ -n "$name" ] || name=gomad3
	"$tgo" doc -all "$package" > "$out/go-doc/$name.txt" 2>&1 ||
		fail "go doc failed for $package (see $out/go-doc/$name.txt)"
done < "$out/go-doc/packages.txt"

cd "$top"
"$tgo" doc -all ./tools/gomad3sim > "$out/go-doc/gomad3sim.txt" 2>&1 ||
	fail "go doc failed for ./tools/gomad3sim (see $out/go-doc/gomad3sim.txt)"

# CLI surface, from binaries built out of the current source.
cd "$module"
"$tgo" build -trimpath -o "$bin/gomad" ./cmd/gomad
"$tgo" build -trimpath -o "$bin/gomadtool" ./cmd/gomadtool

capture() {
	name=$1
	shift
	tool=$1
	shift
	if "$bin/$tool" "$@" > "$bin/capture.stdout" 2> "$bin/capture.stderr" < /dev/null; then
		status=0
	else
		status=$?
	fi
	{
		printf '$ %s' "$tool"
		for argument in "$@"; do printf ' %s' "$argument"; done
		printf '\nexit status: %s\n--- stdout\n' "$status"
		cat "$bin/capture.stdout"
		printf '%s\n' '--- stderr'
		cat "$bin/capture.stderr"
	} > "$out/cli/$name.txt"
}

capture gomad gomad
capture gomad_--help gomad --help
for command in plan execute-shard merge explore qualify qualify-set merge-set \
	compare-support analyze resume recover replay minimize doctor inspect; do
	capture "gomad_$command" gomad "$command" --help
done

capture gomadtool gomadtool
capture gomadtool_--help gomadtool --help
for command in boundary-generate build-key checked-run compatibility-pack \
	patch-materialize patch-regenerate patch-validate protocol-generate \
	qualification-manifest-generate script-validate test toolchain-build \
	upgrade-dossier version-generate; do
	capture "gomadtool_$command" gomadtool "$command" --help
done
capture gomadtool_compatibility-pack_bare gomadtool compatibility-pack
for command in discover review generate check qualify; do
	capture "gomadtool_compatibility-pack_$command" gomadtool compatibility-pack "$command" --help
done

echo "api-capture: wrote $out"
