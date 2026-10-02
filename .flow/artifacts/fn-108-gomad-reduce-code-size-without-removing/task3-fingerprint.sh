#!/bin/sh
# fn-108.3 fingerprint runner. Read-only for the repository: the probe test is
# compiled in through `go test -overlay` and never written into the checkout.
#
# usage: fingerprint.sh PROBE_TEST_GO OUT_PREFIX
#   PROBE_TEST_GO  absolute path of the probe test source (outside the repository)
#   OUT_PREFIX     absolute path prefix; receives OUT_PREFIX, OUT_PREFIX.failures,
#                  OUT_PREFIX.mmap_unix.go, OUT_PREFIX.log and OUT_PREFIX.overlay.json
set -eu
[ "$#" -eq 2 ] || { echo "usage: fingerprint.sh PROBE_TEST_GO OUT_PREFIX" >&2; exit 2; }
probe=$1
out=$2
case "$probe" in /*) ;; *) echo "PROBE_TEST_GO must be absolute" >&2; exit 2 ;; esac
case "$out" in /*) ;; *) echo "OUT_PREFIX must be absolute" >&2; exit 2 ;; esac
top=$(cd "$(git rev-parse --show-toplevel)" && pwd -P)
module="$top/tools/gomad3"
printf '{"Replace":{"%s/deterministicio/zz_fn108_task3_fingerprint_test.go":"%s"}}\n' "$module" "$probe" > "$out.overlay.json"
cd "$module"
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off FN108_FINGERPRINT_OUT="$out" \
	.toolchain/bin/go test -count=1 -tags test_dep -overlay "$out.overlay.json" \
	-run '^TestFn108Task3Fingerprint$' -v ./deterministicio > "$out.log" 2>&1
