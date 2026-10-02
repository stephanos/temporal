#!/usr/bin/env bash
# Elaborate the copied standalone activity Model against the built production model and write its
# parity dumps. Elaboration runs the Lean command DSL, so this takes minutes.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
out="$here/../parity/testdata/lean"
cd "$here/../../lean"
if [[ "$(uname -s)" == Darwin ]]; then
  export SDKROOT="$(xcrun --show-sdk-path)" CC="$(xcrun --find clang)" CXX="$(xcrun --find clang++)"
  PATH="$(dirname "$CC"):$PATH"
fi
mise exec -- lake build Temporal.Case.Syntax Temporal.Feature.Worker.Model
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mise exec -- lake env lean --run "$here/ActivityDump.lean" "$tmp"
[[ -n "$(ls -A "$tmp")" ]] || { echo "activity-dump.sh: nothing written" >&2; exit 1; }
rm -f "$out"/activity-*.json
cp "$tmp"/*.json "$out"/
