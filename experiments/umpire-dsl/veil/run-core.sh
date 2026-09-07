#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
revision=be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd
source_dir=.lake/packages/veil
if [[ ! -d "$source_dir/.git" ]]; then
  mkdir -p .lake/packages
  git clone --filter=blob:none --no-checkout https://github.com/verse-lab/veil.git "$source_dir"
  git -C "$source_dir" checkout "$revision"
fi
[[ "$(git -C "$source_dir" rev-parse HEAD)" == "$revision" ]]
git -C "$source_dir" diff --exit-code "$revision" -- Veil/Core/Tools/ModelChecker/ExecutionOutcome.lean Veil/Core/Tools/ModelChecker/TransitionSystem.lean
mkdir -p .lake/core/Veil/Core/Tools/ModelChecker .lake/core/DslExperiment
lean -R .. -o .lake/core/DslExperiment/Model.olean ../DslExperiment/Model.lean
lean -o .lake/core/Veil/Core/Tools/ModelChecker/ExecutionOutcome.olean "$source_dir/Veil/Core/Tools/ModelChecker/ExecutionOutcome.lean"
LEAN_PATH=.lake/core lean -o .lake/core/Veil/Core/Tools/ModelChecker/TransitionSystem.olean "$source_dir/Veil/Core/Tools/ModelChecker/TransitionSystem.lean"
LEAN_PATH=.lake/core lean --run Main.lean
