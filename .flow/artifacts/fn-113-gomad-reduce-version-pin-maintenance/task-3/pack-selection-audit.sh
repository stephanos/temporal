#!/bin/sh
# Pack selection audit. For every Go module tracked in the repository, run
# gomadtool pin-impact with that module as the baseline and an empty module as
# the candidate. Every pack rule the baseline selects is then reported "stale"
# (the candidate no longer requires its module), so the stale pins list
# exactly the packs each module selects. Prints "<module dir> <pack id>" lines.
set -eu
repo=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
out=${1:?output directory}
mkdir -p "$out/empty"
printf 'module gomad3.pack-selection-audit.empty\n\ngo 1.27.0\n' >"$out/empty/go.mod"
cd "$repo/tools/gomad3"
go build -o "$out/gomadtool" ./cmd/gomadtool
git -C "$repo" ls-files | grep -E '(^|/)go\.mod$' | grep -v '^\.flow/' | while read -r mod; do
  dir=$(dirname "$mod")
  name=$(echo "$dir" | tr '/' '_')
  status=0
  "$out/gomadtool" pin-impact --root=. --baseline-module="$repo/$dir" --module="$out/empty" --json >"$out/$name.json" 2>"$out/$name.err" || status=$?
  case $status in 0|1) ;; *) echo "pin-impact failed for $dir (status $status):" >&2; cat "$out/$name.err" >&2; exit 1;; esac
  jq -r --arg dir "$dir" '.pins[]|select(.class=="pack-rule")|if .status=="stale" then "\($dir) \(.pack)" else error("unexpected pack-rule status \(.status) for \(.id)") end' "$out/$name.json" | sort -u
  jq -r --arg dir "$dir" '"\($dir) (pack rules selected: \([.pins[]|select(.class=="pack-rule" and .status=="stale")]|length))"' "$out/$name.json"
done
