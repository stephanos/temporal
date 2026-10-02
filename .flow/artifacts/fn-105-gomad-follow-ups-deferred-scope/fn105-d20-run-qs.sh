#!/bin/bash
# Runs `gomad qualify-set` for a scratch manifest and prints the per-seed classifications.
#
# usage: D20_OUT=<scratch dir> fn105-d20-run-qs.sh <name> <manifest> [source-dir]
#
# The report is written to $D20_OUT/<name>-report.json and the log to $D20_OUT/<name>.log.
# Artifacts go to $D20_OUT/<name>-qs-artifacts and are removed through the scratchpad's
# safe-rm.sh afterwards. [source-dir] defaults to the repository that holds this script.
set -euo pipefail
HERE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
SOURCE_ROOT=${D20_REPO:-$(cd -- "$HERE/../../.." && pwd -P)}
SCRATCHPAD=/private/tmp/claude-501/-Users-stephan-Workspace-temporal-gomad/f662b53b-adef-414e-85f5-25221ab92e04/scratchpad
SAFE_RM=${D20_SAFE_RM:-$SCRATCHPAD/safe-rm.sh}
OUT=${D20_OUT:?set D20_OUT to a directory inside the scratchpad}
case "$OUT" in "$SCRATCHPAD"/?*) ;; *) echo "D20_OUT must be inside $SCRATCHPAD" >&2; exit 2;; esac
[ "$#" -ge 2 ] || { echo "usage: $0 <name> <manifest> [source-dir]" >&2; exit 2; }
name=$1; manifest=$2; dir=${3:-$SOURCE_ROOT}
case "$name" in *[!A-Za-z0-9_.-]*|"") echo "bad name: $name" >&2; exit 2;; esac
export PATH="$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH"
art="$OUT/$name-qs-artifacts"
if [ -e "$art" ]; then "$SAFE_RM" "$art"; fi
start=$(date +%s)
rc=0
env -u GOMADSEED -u GOMAD3_CHILD_SEED "$SOURCE_ROOT/tools/gomad3/.bin/gomad" qualify-set --manifest="$manifest" \
  --working-dir="$dir" --artifacts="$art" --output="$OUT/$name-report.json" --prune-qualified-artifacts \
  > "$OUT/$name.log" 2>&1 || rc=$?
echo "name=$name exit=$rc wall=$(( $(date +%s) - start ))s report=$OUT/$name-report.json"
tail -5 "$OUT/$name.log" | cut -c1-300
if [ -e "$art" ]; then "$SAFE_RM" "$art"; fi
