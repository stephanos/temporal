#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
probe_root="$PWD"
[[ -f .lake/core/Veil/Core/Tools/ModelChecker/TransitionSystem.olean ]] || {
  echo 'Run ./run-core.sh first to prepare the pinned semantic core.' >&2
  exit 1
}
edit_dir=$(mktemp -d "$probe_root/.lake/edit-probe.XXXXXX")
trap 'rm -rf "$edit_dir"' EXIT
mkdir -p "$edit_dir/DslExperiment"
cp ../DslExperiment/Model.lean "$edit_dir/DslExperiment/Model.lean"
cp Main.lean "$edit_dir/Main.lean"
export LEAN_PATH="$edit_dir:$probe_root/.lake/core"
lean -R "$edit_dir" -o "$edit_dir/DslExperiment/Model.olean" "$edit_dir/DslExperiment/Model.lean"
lean -R "$edit_dir" --run "$edit_dir/Main.lean"
python3 - "$edit_dir/DslExperiment/Model.lean" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
source = path.read_text()
anchor = '  | .started, .requested => some .requested\n'
assert source.count(anchor) == 1
addition = '  | .started, .completed => some .succeeded\n'
assert addition not in source
path.write_text(source.replace(anchor, anchor + addition))
PY
lean -R "$edit_dir" -o "$edit_dir/DslExperiment/Model.olean" "$edit_dir/DslExperiment/Model.lean"
lean -R "$edit_dir" -o "$edit_dir/Main.olean" "$edit_dir/Main.lean"
lean EditProbe.lean
cmp Main.lean "$edit_dir/Main.lean"
