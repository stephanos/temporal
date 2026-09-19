#!/bin/sh
set -eu

fixture_source=$(CDPATH='' cd -- "$(dirname -- "$0")/testdata" && pwd)
fixture_build=$(mktemp -d "${TMPDIR:-/tmp}/umpire-lean-api.XXXXXX")
trap 'rm -rf "$fixture_build"' EXIT HUP INT TERM

for fixture in basic empty-service; do
    case "$fixture" in
        basic) root=Fixture ;;
        empty-service) root=EmptyFixture ;;
    esac
    (
        cp -R "$fixture_source/$fixture/expected/$root" "$fixture_build/$root"
        cd "$fixture_build"
        export LEAN_PATH="$fixture_build:$LEAN_PATH"
        lean -o "$root/API/Proto.olean" "$root/API/Proto.lean"
        lean -o "$root/API/Types.olean" "$root/API/Types.lean"
        lean -o "$root/API.olean" "$root/API.lean"
        lean "$fixture_source/$fixture/checks.lean"
    )
done
lean "$fixture_source/temporal_checks.lean"
