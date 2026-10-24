#!/usr/bin/env bash
# Rerun the native symbolic back end spike. Needs: a ric3 binary (RIC3), optionally a z3 binary
# (SMT_SOLVER), and the model/umpire crates (dsl, ir, checker) beside this tree.
#
#   RIC3=/path/to/ric3 SMT_SOLVER=/path/to/z3 ./run.sh            # baseline + controls
#   RIC3=/path/to/ric3 RUNS=big ./run.sh                          # scaled Kettle (needs a big stack)
#
# Building ric3 from source (https://github.com/gipsyh/rIC3, v1.5.2 was used): it needs cmake and
# a `clang` on PATH; `pip install --target <dir> cmake` and a `clang -> gcc` symlink were enough.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
: "${RIC3:?set RIC3 to the ric3 binary}"
export RUNS="${RUNS:-baseline,claim99,heat2e9}"
export ENGINES="${ENGINES:-ic3,kind,bmc}"
export TIMEOUT="${TIMEOUT:-60}"
export OUT="${OUT:-$here/out}"
ulimit -s unlimited 2>/dev/null || true   # the native checker's 2M-state exploration on `big`
cd "$here"
cargo +1.99.1 build --release --quiet
exec "${CARGO_TARGET_DIR:-target}/release/umpire-native-symbolic"
