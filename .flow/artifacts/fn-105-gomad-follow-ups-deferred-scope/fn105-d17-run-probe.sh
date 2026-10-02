#!/bin/bash
# usage: run-probe.sh NAME [ENV_ENTRY...]  -> runs the envprobe program under gomad explore; never deletes.
set -euo pipefail
D=/private/tmp/claude-501/-Users-stephan-Workspace-temporal-gomad/f662b53b-adef-414e-85f5-25221ab92e04/scratchpad/d17
GOMAD=/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.bin/gomad
name="$1"; shift
case "$name" in *[!A-Za-z0-9._-]*|"") echo "bad name" >&2; exit 2;; esac
[ ! -e "$D/art/$name" ] || { echo "exists" >&2; exit 2; }
export PATH="$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH"
args=(explore --seeds 1)
for e in "$@"; do args+=("--env=$e"); done
args+=(--artifacts "$D/art/$name" go-run . --)
cd "$D/envprobe"
rc=0
env -u GOMADSEED -u GOMAD3_CHILD_SEED "$GOMAD" "${args[@]}" > "$D/out/$name.gomad.log" 2>&1 || rc=$?
echo "probe=$name supplied=$* exit=$rc"
for d in "$D/art/$name"/v1/campaign-*/failures/*; do
  echo "target stdout: $(cat "$d/stdout")"
  python3 -c "import json,sys;m=json.load(open(sys.argv[1]));print('recorded environment:',[e['name']+'='+e['value'] for e in m['environment']])" "$d/manifest.json"
done
