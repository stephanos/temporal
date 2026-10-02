#!/bin/bash
# usage: run-qs.sh NAME MANIFEST_BASENAME WORKDIR
# Runs gomad qualify-set once. Never deletes anything: NAME must be new.
set -euo pipefail
D=/private/tmp/claude-501/-Users-stephan-Workspace-temporal-gomad/f662b53b-adef-414e-85f5-25221ab92e04/scratchpad/d17
GOMAD=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.bin/gomad
[ "$#" -eq 3 ] || { echo "usage: run-qs.sh NAME MANIFEST_BASENAME WORKDIR" >&2; exit 2; }
name="$1"; manifest="$D/m/$2"; workdir="$3"
case "$name" in *[!A-Za-z0-9._-]*|"") echo "bad name" >&2; exit 2;; esac
[ -f "$manifest" ] || { echo "no manifest $manifest" >&2; exit 2; }
[ -d "$workdir/tests" ] || { echo "no workdir $workdir" >&2; exit 2; }
[ ! -e "$D/art/$name" ] || { echo "artifacts dir exists, pick a new name" >&2; exit 2; }
[ ! -e "$D/out/$name" ] || { echo "output dir exists, pick a new name" >&2; exit 2; }
export PATH="$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH"
mkdir -p "$D/out/$name"
start=$(date +%s)
cd "$workdir"
rc=0
env -u GOMADSEED -u GOMAD3_CHILD_SEED "$GOMAD" qualify-set "--manifest=$manifest" "--working-dir=$workdir" "--artifacts=$D/art/$name" "--output=$D/out/$name/report.json" > "$D/out/$name/qualify-set.log" 2>&1 || rc=$?
echo "name=$name manifest=$2 workdir=$workdir exit=$rc wall=$(( $(date +%s) - start ))s load=$(uptime | sed 's/.*averages: //')"
tail -2 "$D/out/$name/qualify-set.log" | cut -c1-300
if [ -d "$D/art/$name/qualifications/v1" ]; then cp "$D/art/$name/qualifications/v1/"*.json "$D/out/$name/"; fi
