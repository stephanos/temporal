#!/usr/bin/env bash
# The backend agreement gate: exports the lifted IR to Quint and one monitor to P, runs both tools,
# and holds each to Go's reading of the same Models. It fails when a tool is missing, when a tool
# fails, and on any disagreement; nothing is skipped.
#
#   model/scalav2/backends/run.sh              run the agreement; print one receipt per comparison
#   model/scalav2/backends/run.sh --install    first install .NET 8 and P 3.1.0 into the tool directory
#   model/scalav2/backends/run.sh --out DIR    keep every export, dump and report under DIR
#
# Tools, none of which the repository's mise.toml or the default `go test` needs:
#   Quint 0.33.0   through model/quint/quint.sh, which runs that exact version from npm's cache
#   Apalache       0.62.1, which `quint verify` downloads into ~/.quint; a JVM, the repository's
#   P 3.1.0        a .NET tool, in ${UMPIRE_BACKEND_TOOLS:-.build/umpire-backend-tools}/p
#   .NET SDK 8.0   in ${UMPIRE_BACKEND_TOOLS:-.build/umpire-backend-tools}/dotnet
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../../.." && pwd)"
# Under .build, which git ignores and a restart keeps; /tmp does not survive one.
tools="${UMPIRE_BACKEND_TOOLS:-$root/.build/umpire-backend-tools}"
quint_version=0.33.0
p_version=3.1.0
cd "$root"

install=false
out=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --install) install=true ;;
    --out) out="${2:?--out takes a directory}"; shift ;;
    *) echo "usage: run.sh [--install] [--out DIR]" >&2; exit 2 ;;
  esac
  shift
done

export DOTNET_ROOT="$tools/dotnet" DOTNET_CLI_TELEMETRY_OPTOUT=1 DOTNET_NOLOGO=1
export PATH="$tools/dotnet:$PATH"

if "$install"; then
  mkdir -p "$tools"
  if [[ ! -x "$tools/dotnet/dotnet" ]]; then
    curl -sSL https://dot.net/v1/dotnet-install.sh -o "$tools/dotnet-install.sh"
    bash "$tools/dotnet-install.sh" --channel 8.0 --install-dir "$tools/dotnet"
  fi
  if [[ ! -x "$tools/p/p" ]]; then
    "$tools/dotnet/dotnet" tool install P --version "$p_version" --tool-path "$tools/p"
  fi
fi

missing() { echo "umpire-backends: $1" >&2; exit 1; }

quint="$root/model/quint/quint.sh"
command -v npm > /dev/null || missing "npm is not on the path: Quint $quint_version runs from npm's cache"
[[ "$("$quint" --version)" == "$quint_version" ]] || missing "$quint does not run Quint $quint_version"
mise exec -- java -version > /dev/null 2>&1 || missing "no JVM: \`quint verify\` runs Apalache on the repository's JDK (mise.toml)"
[[ -x "$tools/p/p" ]] || missing "P is not installed in $tools/p: run model/scalav2/backends/run.sh --install"
"$tools/p/p" --version 2>&1 | grep -q "P version ${p_version}\." \
  || missing "$tools/p/p is not P $p_version: remove $tools/p and run with --install"

export UMPIRE_QUINT="$quint" UMPIRE_P="$tools/p/p" UMPIRE_BACKENDS=require
if [[ -n "$out" ]]; then
  mkdir -p "$out"
  UMPIRE_BACKENDS_OUT="$(cd "$out" && pwd)"
  export UMPIRE_BACKENDS_OUT
fi

log="$(mktemp)"
# The Apalache server `quint verify` starts listens on a port of this gate's own, and is stopped with it.
trap 'rm -f "$log"; pkill -f "apalache.jar server --port=38822" 2> /dev/null || true' EXIT
status=0
CC="${CC:-/usr/bin/clang}" mise exec -- go test -tags test_dep -count=1 -v ./model/scalav2/backends/ > "$log" 2>&1 || status=$?
sed -n 's/^.*RECEIPT //p' "$log" | sort -u
if [[ -n "$out" ]]; then
  sed -n 's/^.*RECEIPT //p' "$log" | sort -u > "$out/receipts.txt"
  cp "$log" "$out/go-test.log"
fi
if [[ "$status" -ne 0 ]]; then
  grep -E '^(---|\s+---) FAIL|^FAIL|Error:|Error Trace:' "$log" >&2 || cat "$log" >&2
  missing "the backend agreement failed"
fi
if grep -q -- '--- SKIP' "$log"; then
  grep -- '--- SKIP' "$log" >&2
  missing "a comparison was skipped, and a comparison that did not run is no agreement"
fi
echo "umpire-backends: Quint $quint_version and P $p_version agree with Go; see the receipts above for what was compared and what was not"
