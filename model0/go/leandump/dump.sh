#!/usr/bin/env bash
# Refresh the Lean parity dumps from the built production model. Needs `lake build` in model/.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
out="$here/../parity/testdata/lean"
mkdir -p "$out"
cd "$here/../../lean"
if [[ "$(uname -s)" == Darwin ]]; then
  export SDKROOT="$(xcrun --show-sdk-path)" CC="$(xcrun --find clang)" CXX="$(xcrun --find clang++)"
  PATH="$(dirname "$CC"):$PATH"
fi
mise exec -- lake build Temporal.Feature.Nexus.Caller.Model
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
# Write into a scratch directory first: a Lean error must fail the script and leave the committed
# dumps untouched rather than half-rewritten.
mise exec -- lake env lean --run "$here/Dump.lean" "$tmp"
[[ -n "$(ls -A "$tmp")" ]] || { echo "dump.sh: Dump.lean wrote nothing" >&2; exit 1; }
find "$out" -maxdepth 1 -type f ! -name "activity-*" -delete
cp "$tmp"/*.json "$tmp"/*.txt "$out"/
