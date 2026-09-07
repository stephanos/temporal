#!/usr/bin/env bash
set -euo pipefail
experiment_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
model_dir="$(cd -- "$experiment_dir/../../model" && pwd)"
cd -- "$model_dir"
model_lean_path="$(lake env printenv LEAN_PATH)"
export LEAN_PATH="$experiment_dir/.lake/build/lib/lean:$model_lean_path"
exec lake env lean --run "$experiment_dir/Baseline.lean"
